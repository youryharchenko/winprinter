package main

import (
	"flag"
	"log"

	"github.com/youryharchenko/winprinter/winspool"
)

type Job struct {
	file  string
	title string
}

var file = flag.String("file", "cells.ps", "print file")

//var printerName = flag.String("printer", "Foxit Reader PDF Printer", "printer name")
//var printerName = flag.String("printer", "Microsoft XPS Document Writer", "printer name")
var printerName = flag.String("printer", "EPSON BA-T500 Receipt", "printer name")

func main() {
	flag.Parse()
	log.Println("Start Winprint!")

	job := NewJob(*file, *file)
	if printer, err1 := winspool.NewPrinter(*printerName); err1 != nil {
		log.Fatalln(err1)
	} else {
		if id, err2 := job.Print(printer); err2 != nil {
			log.Fatalln(err2)
		} else {
			ret, msg, stat1, stat2 := printer.GetJobStatus(id)
			log.Println("job:", id, "ret:", ret, "msg:", msg, "stat1:", stat1, "stat2:", stat2)
			if err3 := printer.Close(); err3 != nil {
				log.Fatalln(err3)
			} else {
				log.Println("OK")
			}
		}
	}
}

func NewJob(file string, title string) (job *Job) {
	job = new(Job)
	job.file = file
	job.title = title
	return
}

func (job *Job) Print(printer *winspool.Printer) (jobId uint32, err error) {
	if jobId, err = printer.PrintPostScriptFile(job.file, job.title); err != nil {
		log.Fatalln(err)
	}
	return
}
