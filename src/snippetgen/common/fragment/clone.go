package fragment

import (
	"fmt"
	"log"

	"discovery-artifact-manager/common/clone"
)

// Clone creates a deep copy of 'info'.
func (info *Info) Clone() *Info {
	var cloned *Info
	if err := clone.Clone(info, &cloned); err != nil {
		log.Printf(fmt.Sprintf("error in Info.Clone(): %s", err))
		return nil
	}
	return cloned
}

// Clone creates a deep copy of 'path'.
func (path *Path) Clone() *Path {
	var cloned *Path
	if err := clone.Clone(path, &cloned); err != nil {
		log.Printf(fmt.Sprintf("error in Path.Clone(): %s", err))
		return nil
	}
	return cloned
}

// Clone creates a deep copy of 'file'.
func (file *File) Clone() *File {
	var cloned *File
	if err := clone.Clone(file, &cloned); err != nil {
		log.Printf(fmt.Sprintf("error in File.Clone(): %s", err))
		return nil
	}
	return cloned
}

// Clone creates a deep copy of 'code'.
func (code *CodeFragment) Clone() *CodeFragment {
	var cloned *CodeFragment
	if err := clone.Clone(code, &cloned); err != nil {
		log.Printf(fmt.Sprintf("error in CodeFragment.Clone(): %s", err))
		return nil
	}
	return cloned
}
