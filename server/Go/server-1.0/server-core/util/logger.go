package util

/************************************************************************************

Logrus has seven logging levels: Trace, Debug, Info, Warning, Error, Fatal and Panic.
Ex:
 log.Error("read: ", err)
 ...
 log.WithFields(log.Fields{
		"Method": r.Method,
		"Host":r.Host,
		"Proto":r.Proto,
		"Uri":r.RequestURI,
	}).Info("http request fields")
...
For more info : https://github.com/sirupsen/logrus/blob/master/README.md

***********************************************************************************/

import (
	"os"
	log "github.com/sirupsen/logrus"
)


func InitLogger(){
	log.SetFormatter(&log.JSONFormatter{})

	// Output to stdout instead of the default stderr
	// Can be any io.Writer, see below for File example
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.TraceLevel)
}
