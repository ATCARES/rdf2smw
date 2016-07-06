// Workflow written in SciPipe.
// For more information about SciPipe, see: http://scipipe.org
package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/d4l3k/turtle"
	"github.com/flowbase/flowbase"
	"log"
	"os"
	str "strings"
)

const (
	BUFSIZE = 16
)

func main() {
	flowbase.InitLogInfo()

	inFileName := flag.String("infile", "", "The input file name")
	flag.Parse()
	if *inFileName == "" {
		fmt.Println("No filename specified to --infile")
		os.Exit(1)
	}

	// Create a pipeline runner
	pipeRunner := flowbase.NewPipelineRunner()

	// Initialize processes and add to runner
	fileReader := NewFileReader()
	pipeRunner.AddProcess(fileReader)

	//tripleParser := NewTripleParser()
	//pipeRunner.AddProcess(tripleParser)

	turtleToTriples := NewTurtleToTriples()
	pipeRunner.AddProcess(turtleToTriples)

	triplePrinter := NewTriplePrinter()
	pipeRunner.AddProcess(triplePrinter)

	// Connect workflow dependency network
	turtleToTriples.In = fileReader.OutLine
	triplePrinter.In = turtleToTriples.Out

	// Run the pipeline!
	go func() {
		defer close(fileReader.InFileName)
		fileReader.InFileName <- *inFileName
	}()

	pipeRunner.Run()

}

// ================================================================================
// Components
// ================================================================================

// --------------------------------------------------------------------------------
// FileReader
// --------------------------------------------------------------------------------

type FileReader struct {
	InFileName chan string
	OutLine    chan string
}

func NewFileReader() *FileReader {
	return &FileReader{
		InFileName: make(chan string, BUFSIZE),
		OutLine:    make(chan string, BUFSIZE),
	}
}

func (p *FileReader) Run() {
	defer close(p.OutLine)

	flowbase.Debug.Println("Starting loop")
	for fileName := range p.InFileName {
		flowbase.Debug.Printf("Starting processing file %s\n", fileName)
		fh, err := os.Open(fileName)
		if err != nil {
			log.Fatal(err)
		}
		defer fh.Close()

		sc := bufio.NewScanner(fh)
		for sc.Scan() {
			if err := sc.Err(); err != nil {
				log.Fatal(err)
			}
			p.OutLine <- sc.Text()
		}
	}
}

// --------------------------------------------------------------------------------
// TripleParser
// --------------------------------------------------------------------------------

type TripleParser struct {
	In  chan string
	Out chan *RDFTriple
}

func NewTripleParser() *TripleParser {
	return &TripleParser{
		In:  make(chan string, BUFSIZE),
		Out: make(chan *RDFTriple, BUFSIZE),
	}
}

func (p *TripleParser) Run() {
	defer close(p.Out)
	for line := range p.In {
		triple := NewRDFTriple()
		bits := str.Split(line, " ")
		triple.Subject = bits[0]
		triple.Predicate = bits[1]
		triple.Object = bits[2]
		p.Out <- triple
	}
}

// --------------------------------------------------------------------------------
// TurtleToTriples
// --------------------------------------------------------------------------------

type TurtleToTriples struct {
	In  chan string
	Out chan turtle.Triple
}

func NewTurtleToTriples() *TurtleToTriples {
	return &TurtleToTriples{
		In:  make(chan string, BUFSIZE),
		Out: make(chan turtle.Triple, BUFSIZE),
	}
}

func (p *TurtleToTriples) Run() {
	defer close(p.Out)
	for line := range p.In {
		triples, err := turtle.Parse([]byte(line))
		if err != nil {
			flowbase.Error.Printf("Error parsing turtle: %v", err)
			os.Exit(1)
		}
		for _, triple := range triples {
			p.Out <- triple
		}
	}
}

// --------------------------------------------------------------------------------
// TriplePrinter
// --------------------------------------------------------------------------------

type TriplePrinter struct {
	In chan turtle.Triple
}

func NewTriplePrinter() *TriplePrinter {
	return &TriplePrinter{
		In: make(chan turtle.Triple, BUFSIZE),
	}
}

func (p *TriplePrinter) Run() {
	for tr := range p.In {
		fmt.Printf("S: %s\nP: %s\nO: %s\n\n", tr.Subj, tr.Pred, tr.Obj)
	}
}

// --------------------------------------------------------------------------------
// IP: RDFTriple
// --------------------------------------------------------------------------------

type RDFTriple struct {
	Subject   string
	Predicate string
	Object    string
}

func NewRDFTriple() *RDFTriple {
	return &RDFTriple{}
}
