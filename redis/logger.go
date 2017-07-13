package redis


import (
	"log"
	"os"
)

var logger = log.New(os.Stdout, "", log.Ldate | log.Ltime | log.Llongfile)