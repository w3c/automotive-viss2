// redisInit must be started with root permission (sudo ./redisInit)
// !!!!! redisInit must be executed before running serverclient or feederclient !!!!

package main

import (
	"fmt"
	"os/exec"
//	"github.com/go-redis/redis"
	"github.com/go-redis/redis/v8"
	"bytes"
	"context"
)


func main() {
    client := redis.NewClient(&redis.Options{
        Network:  "unix",
        Addr:     "/var/tmp/vissv2/redisDB.sock",
        Password: "",
        DB:       1,
    })
    ctx := context.TODO()  //redis/v8
//    err := client.Ping().Err()
    err := client.Ping(ctx).Err() //redis/v8
    if err != nil {
//        out, err := exec.Command("redis-server", "/etc/redis/redis.conf").Output()
        redisStartCmd := exec.Command("redis-server", "/etc/redis/redis.conf")
//        if err != nil {
//            fmt.Printf("Starting redis server failed. Err=%s\n", err)
var out bytes.Buffer
var stderr bytes.Buffer
redisStartCmd.Stdout = &out
redisStartCmd.Stderr = &stderr
err := redisStartCmd.Run()
if err != nil {
    fmt.Println(fmt.Sprint(err) + ": " + stderr.String())
}
fmt.Println("Result: " + out.String())
//        } else {
//            fmt.Printf("Redis server started.%s\n", redisStartCmd)
//        }
    } else {
            fmt.Printf("Redis server is running.\n")
    }

}
