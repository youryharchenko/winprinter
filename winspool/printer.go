package winspool

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"syscall"
	"unsafe"
)

var winspool = syscall.NewLazyDLL("Winspool.drv")
var openPrinter = winspool.NewProc("OpenPrinterW")
var startDocPrinter = winspool.NewProc("StartDocPrinterW")
var startPagePrinter = winspool.NewProc("StartPagePrinter")
var writePrinter = winspool.NewProc("WritePrinter")
var endPagePrinter = winspool.NewProc("EndPagePrinter")
var endDocPrinter = winspool.NewProc("EndDocPrinter")
var closePrinter = winspool.NewProc("ClosePrinter")
var getJob = winspool.NewProc("GetJobA") //GetJobW

type docInfo struct {
	pDocName    uintptr
	pOutputFile uintptr
	pDatatype   uintptr
}

type jobInfo struct {
	jobId        uint32
	pPrinterName uintptr
	pMachineName uintptr
	pUserName    uintptr
	pDocument    uintptr
	pDatatype    uintptr
	pStatus      uintptr
	status       uint32
	priority     uint32
	position     uint32
	totalPages   uint32
	pagesPrinted uint32
	submitted    systemTime
}

type systemTime struct {
	wYear         uint16
	wMonth        uint16
	wDayOfWeek    uint16
	wDay          uint16
	wHour         uint16
	wMinute       uint16
	wSecond       uint16
	wMilliseconds uint16
}

//typedef struct _JOB_INFO_1 {
//  DWORD      JobId;
//  LPTSTR     pPrinterName;
//  LPTSTR     pMachineName;
//  LPTSTR     pUserName;
//  LPTSTR     pDocument;
//  LPTSTR     pDatatype;
//  LPTSTR     pStatus;
//  DWORD      Status;
//  DWORD      Priority;
//  DWORD      Position;
//  DWORD      TotalPages;
//  DWORD      PagesPrinted;
//  SYSTEMTIME Submitted;
//}

//typedef struct _SYSTEMTIME {
//  WORD wYear;
//  WORD wMonth;
//  WORD wDayOfWeek;
//  WORD wDay;
//  WORD wHour;
//  WORD wMinute;
//  WORD wSecond;
//  WORD wMilliseconds;
//} SYSTEMTIME, *PSYSTEMTIME;

var jobStatuses = map[int]string{
	0x00000200: "Printer driver cannot print the job.",
	0x00001000: "Job has been delivered to the printer."}

type Printer struct {
	printer syscall.Handle
	name    string
}

func NewPrinter(name string) (printer *Printer, err error) {

	printer = new(Printer)

	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()

	printer.open(name)
	printer.name = name

	// runtime.SetFinalizer(printer, func(p *Printer){
	// 	p.Close()
	// })
	return
}

func (p *Printer) PrintPostScriptFile(path string, title string) (jobId uint32, err error) {

	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()

	log.Println("file:", path)

	defer p.closeDoc()
	jobId = p.openDoc(title)

	defer p.closePage()
	p.openPage()

	p.writeFile(path)

	return
}

func (p *Printer) GetJobStatus(jobId uint32) (uint32, error, string, uint32) {

	var level uint32 = 1
	//var bufSize uint32 = 10000
	var bufSize uint32 = 0
	var realSize uint32
	var job jobInfo

	ret, _, msg := getJob.Call(
		uintptr(unsafe.Pointer(p.printer)),
		uintptr(jobId),
		uintptr(level),
		uintptr(unsafe.Pointer(&job)),
		uintptr(bufSize),
		uintptr(unsafe.Pointer(&realSize)))
	bufSize = realSize

	var job2 jobInfo
	ret, _, msg = getJob.Call(
		uintptr(unsafe.Pointer(p.printer)),
		uintptr(jobId),
		uintptr(level),
		uintptr(unsafe.Pointer(&job2)),
		uintptr(bufSize),
		uintptr(unsafe.Pointer(&realSize)))

	return uint32(ret), msg, string(job2.pStatus), uint32(job2.status)
}

func (p *Printer) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = errors.New(fmt.Sprint(r))
		}
	}()

	p.close()

	return
}

func (p *Printer) open(name string) {

	log.Println("open printer:", name)

	ret, _, msg := openPrinter.Call(
		uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name))),
		uintptr(unsafe.Pointer(&p.printer)),
		uintptr(unsafe.Pointer(nil)))
	if ret != 1 {
		panic(msg)
	}
}

func (p *Printer) close() {

	log.Println("close printer:", p.name)

	ret, _, msg := closePrinter.Call(uintptr(unsafe.Pointer(p.printer)))

	if ret != 1 {
		panic(msg)
	}
}

func (p *Printer) openDoc(name string) (jobId uint32) {
	log.Println("open doc:", name)

	var level uint32 = 1

	var doc docInfo
	doc.pDocName = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr(name)))
	doc.pOutputFile = uintptr(unsafe.Pointer(nil))
	doc.pDatatype = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("RAW")))
	//doc.pDatatype = uintptr(unsafe.Pointer(syscall.StringToUTF16Ptr("XPS_PASS")))

	ret, _, msg := startDocPrinter.Call(
		uintptr(unsafe.Pointer(p.printer)),
		uintptr(level),
		uintptr(unsafe.Pointer(&doc)))
	if ret == 0 {
		panic(msg)
	}

	jobId = uint32(ret)
	return
}

func (p *Printer) closeDoc() {
	log.Println("close doc")

	ret, _, msg := endDocPrinter.Call(uintptr(unsafe.Pointer(p.printer)))
	if ret != 1 {
		panic(msg)
	}
}

func (p *Printer) openPage() {
	log.Println("open page")

	ret, _, msg := startPagePrinter.Call(uintptr(unsafe.Pointer(p.printer)))
	if ret != 1 {
		panic(msg)
	}
}

func (p *Printer) closePage() {
	log.Println("close page")

	ret, _, msg := endPagePrinter.Call(uintptr(unsafe.Pointer(p.printer)))
	if ret != 1 {
		panic(msg)
	}
}

func (p *Printer) writeFile(path string) {

	document, err := ioutil.ReadFile(path)
	if nil != err {
		panic(err)
	}

	log.Println("writeDocument:", string(document))

	var bytesWritten uint32 = 0
	var docSize uint32 = uint32(len(document))

	log.Println("docSize:", docSize)

	ret, _, msg := writePrinter.Call(
		uintptr(unsafe.Pointer(p.printer)),
		uintptr(unsafe.Pointer(&document[0])),
		uintptr(docSize),
		uintptr(unsafe.Pointer(&bytesWritten)))
	if ret != 1 {
		panic(msg)
	}

	log.Println("bytesWritten:", bytesWritten)

}
