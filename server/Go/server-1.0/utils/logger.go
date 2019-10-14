package utils

// cf. https://www.ardanlabs.com/blog/2013/11/using-log-package-in-go.html

import (
    "io"
//    "io/ioutil"
    "log"
    "os"
)

var (
//    Trace   *log.Logger
    Info    *log.Logger
    Warning *log.Logger
    Error   *log.Logger
)

/*
* Possible inparams to Init:
* ioutil.Discard, 
* os.Stdout, 
* os.Stderr)
* utils.LogFile
*/
func InitLog(
//    traceHandle io.Writer,
    infoHandle io.Writer,
    warningHandle io.Writer,
    errorHandle io.Writer) {

/*    Trace = log.New(traceHandle,
        "TRACE: ",
        log.Ldate|log.Ltime|log.Lshortfile)
*/
    Info = log.New(infoHandle,
        "INFO: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    Warning = log.New(warningHandle,
        "WARNING: ",
        log.Ldate|log.Ltime|log.Lshortfile)

    Error = log.New(errorHandle,
        "ERROR: ",
        log.Ldate|log.Ltime|log.Lshortfile)
}

/**
* Must be called before InitLog() if log should be written to log.
* Caller should take responsibility of closing log file before terminating itself.
**/
func InitLogFile(logFileName string) *os.File {
    file, err:= os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        log.Fatalln("Failed to open log file", os.Stdout, ":", err)
        return nil
    }
    return file
}

/**
* The log file is trimmed to 20% of its size when exceeding 10MB.
**/
func TrimLogFile (logFile *os.File) {
    fi, err2 := logFile.Stat()
    if err2 != nil {
        log.Fatalln("Failed to obtain log file stat", os.Stdout, ":", err2)
    }
    if (fi.Size() > 10000000) { // 10 MB
        fout, err3 := os.Create("logtmp.txt")
        if err3 != nil {
            log.Fatalln("Failed to remove untrimmed log file", os.Stdout, ":", err3)
        }
        defer fout.Close()

        _, err4 := logFile.Seek(8000000, io.SeekStart) // trim 8MB
        if err4 != nil {
            log.Fatalln("Failed to open log file", os.Stdout, ":", err4)
        }

        _, err5 := io.Copy(fout, logFile)
        if err5 != nil {
            log.Fatalln("Failed to copy untrimmed parts of log file", os.Stdout, ":", err5)
        }

        if err6 := os.Remove(fi.Name()); err6 != nil {
            log.Fatalln("Failed to remove untrimmed log file", os.Stdout, ":", err6)
        }

        if err7 := os.Rename("logtmp.txt", fi.Name()); err7 != nil {
            log.Fatalln("Failed to rename trimmed log file", os.Stdout, ":", err7)
        }
    }
}
