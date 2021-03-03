//Package libhosty is a pure golang library to manipulate the hosts file
package libhosty

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"runtime"
	"strings"
	"sync"
)

const (
	//Version exposes library version
	Version = "v1.4"

	//UNKNOWN defines unknown line type
	UNKNOWN = 0
	//EMPTY defines empty line type
	EMPTY = 10
	//COMMENT defines comment line type
	COMMENT = 20
	//ADDRESS defines address line type
	ADDRESS = 30

	// defines default path for windows os
	windowsFilePath = "C:\\Windows\\System32\\drivers\\etc\\"
	// defines default path for linux os
	unixFilePath = "/etc/"
	// defines default filename
	hostsFileName = "hosts"
)

//HostsConfig defines parameters to find hosts file.
// FilePath is the absolute path of the hosts file (filename included)
type HostsConfig struct {
	FilePath string
}

//HostsFileLine holds hosts file lines data
type HostsFileLine struct {
	//LineNumber is the original line number
	LineNumber int

	//LineType is one of the types: UNKNOWN, EMPTY, COMMENT, ADDRESS
	LineType int

	//Address is a net.IP representation of the address
	Address net.IP

	//Parts is a slice of the line splitted by '#'
	Parts []string

	//Hostnames is a slice of hostnames for the relative IP
	Hostnames []string

	//Raw is the raw representation of the line, as it is in the hosts file
	Raw string

	//Trimed is a trimed version (no spaces before and after) of the line
	Trimed string

	//Comment is the comment part of the line (if present in an ADDRESS line)
	Comment string

	//IsCommented to know if the current ADDRESS line is commented out (starts with '#')
	IsCommented bool
}

//HostsFile is a reference for the hosts file configuration and lines
type HostsFile struct {
	sync.Mutex

	//Config reference to a HostsConfig object
	Config *HostsConfig

	//HostsFileLines slice of HostsFileLine objects
	HostsFileLines []HostsFileLine
}

//Init returns a new instance of a hostsfile.
// Init takes a HostsConfig object or nil to use the default one
func Init(conf *HostsConfig) (*HostsFile, error) {
	var config *HostsConfig
	var err error

	if conf != nil {
		config = conf
	} else {
		// initialize hostsConfig
		config, err = NewHostsConfig("")
		if err != nil {
			return nil, err
		}
	}

	// allocate a new HostsFile object
	hf := &HostsFile{
		// use default configuration
		Config: config,

		// allocate a new slice of HostsFileLine objects
		HostsFileLines: make([]HostsFileLine, 0),
	}

	// parse the hosts file and load file lines
	hf.HostsFileLines, err = ParseHostsFile(hf.Config.FilePath)
	if err != nil {
		return nil, err
	}

	//return HostsFile
	return hf, nil
}

//NewHostsConfig loads hosts file based on environment.
// NewHostsConfig initialize the default file path based
// on the OS or from a given location if a custom path is provided
func NewHostsConfig(path string) (*HostsConfig, error) {
	// allocate hostsConfig
	var hc *HostsConfig

	// TODO ensure path exists and is a file (not a dir)

	if len(path) > 0 {
		hc = &HostsConfig{
			FilePath: path,
		}
	} else {
		// check OS to load the correct hostsFile location
		if runtime.GOOS == "windows" {
			hc = &HostsConfig{
				FilePath: windowsFilePath + hostsFileName,
			}
		} else if runtime.GOOS == "linux" {
			hc = &HostsConfig{
				FilePath: unixFilePath + hostsFileName,
			}
		} else if runtime.GOOS == "darwin" {
			hc = &HostsConfig{
				FilePath: unixFilePath + hostsFileName,
			}
		} else {
			return nil, fmt.Errorf("Unrecognized OS: %s", runtime.GOOS)
		}
	}

	return hc, nil
}

//GetHostsFileLineByRow returns a ponter to the given HostsFileLine row
func (h *HostsFile) GetHostsFileLineByRow(row int) *HostsFileLine {
	return &h.HostsFileLines[row]
}

//GetHostsFileLineByIP returns the index of the line and a ponter to the given HostsFileLine line
func (h *HostsFile) GetHostsFileLineByIP(ip net.IP) (int, *HostsFileLine) {
	for idx := range h.HostsFileLines {
		if net.IP.Equal(ip, h.HostsFileLines[idx].Address) {
			return idx, &h.HostsFileLines[idx]
		}
	}

	return -1, nil
}

//GetHostsFileLineByAddress returns the index of the line and a ponter to the given HostsFileLine line
func (h *HostsFile) GetHostsFileLineByAddress(address string) (int, *HostsFileLine) {
	ip := net.ParseIP(address)
	return h.GetHostsFileLineByIP(ip)
}

//GetHostsFileLineByHostname returns the index of the line and a ponter to the given HostsFileLine line
func (h *HostsFile) GetHostsFileLineByHostname(hostname string) (int, *HostsFileLine) {
	for idx := range h.HostsFileLines {
		for _, hn := range h.HostsFileLines[idx].Hostnames {
			if hn == hostname {
				return idx, &h.HostsFileLines[idx]
			}
		}
	}

	return -1, nil
}

//RenderHostsFile render and returns the hosts file with the lineFormatter() routine
func (h *HostsFile) RenderHostsFile() string {
	// allocate a buffer for file lines
	var sliceBuffer []string

	// iterate HostsFileLines and popolate the buffer with formatted lines
	for _, l := range h.HostsFileLines {
		sliceBuffer = append(sliceBuffer, lineFormatter(l))
	}

	// strings.Join() prevent the last line from being a new blank line
	// as opposite to a for loop with fmt.Printf(buffer + '\n')
	return strings.Join(sliceBuffer, "\n")
}

//RenderHostsFileLine render and returns the given hosts line with the lineFormatter() routine
func (h *HostsFile) RenderHostsFileLine(row int) string {
	// iterate to find the row to render
	for idx, hfl := range h.HostsFileLines {
		if idx == row {
			return lineFormatter(hfl)
		}
	}

	return ""
}

//SaveHostsFile write hosts file to configured path.
// error is not nil if something goes wrong
func (h *HostsFile) SaveHostsFile() error {
	return h.SaveHostsFileAs(h.Config.FilePath)
}

//SaveHostsFileAs write hosts file to the given path.
// error is not nil if something goes wrong
func (h *HostsFile) SaveHostsFileAs(path string) error {
	// render the file as a byte slice
	dataBytes := []byte(h.RenderHostsFile())

	// write file to disk
	err := ioutil.WriteFile(path, dataBytes, 0644)
	if err != nil {
		return err
	}

	return nil
}

//RemoveRow remove row at given index from HostsFileLines
func (h *HostsFile) RemoveRow(row int) {
	h.Lock()
	defer h.Unlock()

	// prevent out-of-index
	if row < len(h.HostsFileLines) {
		h.HostsFileLines = append(h.HostsFileLines[:row], h.HostsFileLines[row+1:]...)
	}
}

//LookupByHostname check if the given fqdn exists.
// if yes, it returns the index of the address and the associated address.
// error is not nil if something goes wrong
func (h *HostsFile) LookupByHostname(hostname string) (int, net.IP, error) {
	for idx, hfl := range h.HostsFileLines {
		for _, hn := range hfl.Hostnames {
			if hn == hostname {
				return idx, h.HostsFileLines[idx].Address, nil
			}
		}
	}

	return -1, nil, errors.New("Hostname not found")
}

//AddHostRaw add the given ip/fqdn/comment pair
// this is different from AddHost because it does not take care of duplicates
// this just append the new entry to the hosts file
func (h *HostsFile) AddHostRaw(ipRaw, fqdnRaw, comment string) (int, *HostsFileLine, error) {
	// hostname to lowercase
	hostname := strings.ToLower(fqdnRaw)
	// parse ip to net.IP
	ip := net.ParseIP(ipRaw)

	// if we have a valid IP
	if ip != nil {
		// create a new hosts line
		hfl := HostsFileLine{
			LineType:    ADDRESS,
			Address:     ip,
			Hostnames:   []string{hostname},
			Comment:     comment,
			IsCommented: false,
		}

		// append to hosts
		h.HostsFileLines = append(h.HostsFileLines, hfl)

		// get index
		idx := len(h.HostsFileLines) - 1

		// return created entry
		return idx, &h.HostsFileLines[idx], nil
	}

	// return error
	return -1, nil, fmt.Errorf("Cannot parse IP address %s", ipRaw)
}

//AddHost add the given ip/fqdn/comment pair, cleanup is done for previous entry.
// it returns the index of the edited (created) line and a pointer to the hostsfileline object.
// error is not nil if something goes wrong
func (h *HostsFile) AddHost(ipRaw, fqdnRaw, comment string) (int, *HostsFileLine, error) {
	// hostname to lowercase
	hostname := strings.ToLower(fqdnRaw)
	// parse ip to net.IP
	ip := net.ParseIP(ipRaw)

	// if we have a valid IP
	if ip != nil {
		//check if we alredy have the fqdn
		if idx, addr, err := h.LookupByHostname(hostname); err == nil {
			//if actual ip is the same as the given one, we are done
			if net.IP.Equal(addr, ip) {
				// handle comment
				if comment != "" {
					// just replace the current comment with the new one
					h.HostsFileLines[idx].Comment = comment
				}
				return idx, &h.HostsFileLines[idx], nil
			}

			//if address is different, we need to remove the hostname from the previous entry
			for hostIdx, hn := range h.HostsFileLines[idx].Hostnames {
				if hn == hostname {
					if len(h.HostsFileLines[idx].Hostnames) > 1 {
						h.Lock()
						h.HostsFileLines[idx].Hostnames = append(h.HostsFileLines[idx].Hostnames[:hostIdx], h.HostsFileLines[idx].Hostnames[hostIdx+1:]...)
						h.Unlock()
					}

					//remove the line if there are no more hostnames (other than the actual one)
					if len(h.HostsFileLines[idx].Hostnames) <= 1 {
						h.RemoveRow(idx)
					}
				}
			}
		}

		//if we alredy have the address, just add the hostname to that line
		for idx, hfl := range h.HostsFileLines {
			if net.IP.Equal(hfl.Address, ip) {
				h.Lock()
				h.HostsFileLines[idx].Hostnames = append(h.HostsFileLines[idx].Hostnames, hostname)
				h.Unlock()

				// handle comment
				if comment != "" {
					// just replace the current comment with the new one
					h.HostsFileLines[idx].Comment = comment
				}

				// return edited entry
				return idx, &h.HostsFileLines[idx], nil
			}
		}

		// at this point we need to create new host line
		hfl := HostsFileLine{
			LineType:    ADDRESS,
			Address:     ip,
			Hostnames:   []string{hostname},
			Comment:     comment,
			IsCommented: false,
		}

		// generate raw version of the line
		hfl.Raw = lineFormatter(hfl)

		// append to hosts
		h.HostsFileLines = append(h.HostsFileLines, hfl)

		// get index
		idx := len(h.HostsFileLines) - 1

		// return created entry
		return idx, &h.HostsFileLines[idx], nil
	}

	// return error
	return -1, nil, fmt.Errorf("Cannot parse IP address %s", ipRaw)
}

//AddComment adds a new line of type comment with the given comment.
// it returns the index of the edited (created) line and a pointer to the hostsfileline object.
// error is not nil if something goes wrong
func (h *HostsFile) AddComment(comment string) (int, *HostsFileLine, error) {
	h.Lock()
	defer h.Unlock()

	hfl := HostsFileLine{
		LineType: COMMENT,
		Raw:      "# " + comment,
	}

	hfl.Raw = lineFormatter(hfl)

	h.HostsFileLines = append(h.HostsFileLines, hfl)
	idx := len(h.HostsFileLines) - 1
	return idx, &h.HostsFileLines[idx], nil
}

//AddEmpty adds a new line of type empty.
// it returns the index of the edited (created) line and a pointer to the hostsfileline object.
// error is not nil if something goes wrong
func (h *HostsFile) AddEmpty() (int, *HostsFileLine, error) {
	h.Lock()
	defer h.Unlock()

	hfl := HostsFileLine{
		LineType: EMPTY,
	}

	hfl.Raw = ""

	h.HostsFileLines = append(h.HostsFileLines, hfl)
	idx := len(h.HostsFileLines) - 1
	return idx, &h.HostsFileLines[idx], nil
}

//CommentByRow set the IsCommented bit for the given row to true
func (h *HostsFile) CommentByRow(row int) error {
	h.Lock()
	defer h.Unlock()

	if row <= len(h.HostsFileLines) {
		if h.HostsFileLines[row].LineType == ADDRESS {
			if h.HostsFileLines[row].IsCommented != true {
				h.HostsFileLines[row].IsCommented = true
				return nil
			}

			return ErrAlredyCommentedLine
		}

		return ErrNotAnAddressLine
	}

	return ErrUnknown
}

//CommentByIP set the IsCommented bit for the given address to true
func (h *HostsFile) CommentByIP(ip net.IP) error {
	h.Lock()
	defer h.Unlock()

	for idx, hfl := range h.HostsFileLines {
		if net.IP.Equal(ip, hfl.Address) {
			if h.HostsFileLines[idx].IsCommented != true {
				h.HostsFileLines[idx].IsCommented = true
				return nil
			}

			return ErrAlredyCommentedLine
		}

		return ErrAddressNotFound
	}

	return ErrUnknown
}

//CommentByAddress set the IsCommented bit for the given address as string to false
func (h *HostsFile) CommentByAddress(address string) error {
	ip := net.ParseIP(address)

	return h.CommentByIP(ip)
}

//CommentByHostname set the IsCommented bit for the given hostname to true
func (h *HostsFile) CommentByHostname(hostname string) error {
	h.Lock()
	defer h.Unlock()

	for idx := range h.HostsFileLines {
		for _, hn := range h.HostsFileLines[idx].Hostnames {
			if hn == hostname {
				if h.HostsFileLines[idx].IsCommented != true {
					h.HostsFileLines[idx].IsCommented = true
					return nil
				}

				return ErrAlredyCommentedLine
			}
		}

		return ErrHostnameNotFound
	}

	return ErrUnknown
}

//UncommentByRow set the IsCommented bit for the given row to false
func (h *HostsFile) UncommentByRow(row int) error {
	h.Lock()
	defer h.Unlock()

	if row <= len(h.HostsFileLines) {
		if h.HostsFileLines[row].LineType == ADDRESS {
			if h.HostsFileLines[row].IsCommented != false {
				h.HostsFileLines[row].IsCommented = false
				return nil
			}

			return ErrAlredyUncommentedLine
		}

		return ErrNotAnAddressLine
	}

	return ErrUnknown
}

//UncommentByIP set the IsCommented bit for the given address to false
func (h *HostsFile) UncommentByIP(ip net.IP) error {
	h.Lock()
	defer h.Unlock()

	for idx, hfl := range h.HostsFileLines {
		if net.IP.Equal(ip, hfl.Address) {
			if h.HostsFileLines[idx].IsCommented != false {
				h.HostsFileLines[idx].IsCommented = false
				return nil
			}

			return ErrAlredyUncommentedLine
		}

		return ErrNotAnAddressLine
	}

	return ErrUnknown
}

//UncommentByAddress set the IsCommented bit for the given address as string to false
func (h *HostsFile) UncommentByAddress(address string) error {
	ip := net.ParseIP(address)

	return h.UncommentByIP(ip)
}

//UncommentByHostname set the IsCommented bit for the given hostname to false
func (h *HostsFile) UncommentByHostname(hostname string) error {
	h.Lock()
	defer h.Unlock()

	for idx := range h.HostsFileLines {
		for _, hn := range h.HostsFileLines[idx].Hostnames {
			if hn == hostname {
				if h.HostsFileLines[idx].IsCommented != false {
					h.HostsFileLines[idx].IsCommented = false
					return nil
				}

				return ErrAlredyUncommentedLine
			}
		}

		return ErrHostnameNotFound
	}

	return ErrUnknown
}
