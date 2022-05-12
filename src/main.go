package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"errors"

	"github.com/relex/aini"
	"github.com/umputun/go-flags"
)

var opts struct {
	IniFile    string `short:"i" long:"ini" description:"ansible ini file to parse"`
	IniSection string `short:"s" long:"section" description:"ansible ini file section to parse"`
	HostsFile  string `short:"f" long:"file" description:"path to hosts file" required:"false"`
	Tag        string `short:"t" long:"tag" description:"configuration tag" required:"false"`
	Verbose    bool   `short:"v" long:"verbose" description:"verbose output" required:"false"`
}

var project = "unknown"
var revision = "unknown"

func main() {
	fmt.Println(project, revision)

	p := flags.NewParser(&opts, flags.PrintErrors|flags.PassDoubleDash|flags.HelpFlag)
	p.SubcommandsOptional = true
	if _, err := p.Parse(); err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Printf("[ERROR] cli error: %v", err)
		}
		os.Exit(2)
	}

	// fmt.Printf("[DEBUG] options: %+v\n", opts)

	content, err := ioutil.ReadFile(opts.IniFile) // the file is inside the local directory
	if err != nil {
		fmt.Printf("[ERROR] can't read file: %v\n", err)
		os.Exit(2)
	}

	fmt.Printf("Start parse file %s...\n", opts.IniFile)
	// Parse ansible invntory file and extract hosts with ip addresses
	hostsSection, linesCount, err := ansibleToHosts(string(content), opts.IniSection)
	if err != nil {
		log.Printf("[ERROR] inventory parse error: %v\n", err)
		os.Exit(2)
	}

	if linesCount == 0 {
		log.Printf("No addresses found in inventory file\n")
		os.Exit(0)
	}
	fmt.Printf("Parse complete, %d addresses found\n", linesCount)

	// Scan hosts file, look for tagged block and replace it
	srcHostsFile, err := os.Open(opts.HostsFile)
	if err != nil {
		fmt.Println(err)
	}
	defer srcHostsFile.Close()

	dstHostsFileName := opts.HostsFile + "+"
	dstHostsFile, err := os.Create(dstHostsFileName)
	if err != nil {
		fmt.Println(err)
	}
	defer dstHostsFile.Close()

	fileScanner := bufio.NewScanner(srcHostsFile)
	fileScanner.Split(bufio.ScanLines)
	inTaggedSection := false
	sectionWritten := false

	hostsSectionOpenTag := "### TAG: " + opts.Tag + " {{{"
	hostsSectionCloseTag := "### TAG: " + opts.Tag + " }}}"
	for fileScanner.Scan() {
		if !inTaggedSection && fileScanner.Text() == hostsSectionOpenTag {
			inTaggedSection = true
			continue
		}
		if inTaggedSection {
			if fileScanner.Text() == hostsSectionCloseTag {
				inTaggedSection = false

				// write new version of section
				dstHostsFile.WriteString(hostsSectionOpenTag + "\n")
				dstHostsFile.WriteString(hostsSection)
				dstHostsFile.WriteString(hostsSectionCloseTag + "\n")

				sectionWritten = true
			}
			continue // skip line
		}
		dstHostsFile.WriteString(fileScanner.Text() + "\n") // copy line to output file
	}

	if !sectionWritten {
		dstHostsFile.WriteString(hostsSectionOpenTag + "\n")
		dstHostsFile.WriteString(hostsSection)
		dstHostsFile.WriteString(hostsSectionCloseTag + "\n")
	}

	os.Remove(opts.HostsFile)
	os.Rename(dstHostsFileName, opts.HostsFile)

	fmt.Printf("Hosts file generation complete\n")
}

func ansibleToHosts(content string, section string) (string, int, error) {
	inventoryReader := strings.NewReader(content)
	inventory, err := aini.Parse(inventoryReader)
	if err != nil {
		return "", 0, err
	}

	result := ""
	group, ok := inventory.Groups[section]
	if !ok {
		return "", 0, errors.New("Section not found")
	}
	lines := 0
	for _, h := range group.Hosts {
		ip := h.Vars["ansible_host"]
		if strings.TrimSpace(ip) == "" {
			ip = "# no ip"
		}
		result += ip + "\t" + h.Name + "\n"
		lines++
	}

	return result, lines, nil
}
