package main

import (
	"fmt"
	"github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
)

func createShortcut(src, dst string) error {
	// https://stackoverflow.com/questions/32438204/create-a-windows-shortcut-lnk-in-go

	ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED|ole.COINIT_SPEED_OVER_MEMORY)
	oleShellObject, err := oleutil.CreateObject("WScript.Shell")
	if err != nil {
		return fmt.Errorf("CreateObject : %v", err)
	}
	defer oleShellObject.Release()

	wshell, err := oleShellObject.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		return fmt.Errorf("QueryInterface : %v", err)
	}
	defer wshell.Release()

	cs, err := oleutil.CallMethod(wshell, "CreateShortcut", dst)
	if err != nil {
		return fmt.Errorf("CreateShortcut : %v", err)
	}
	idispatch := cs.ToIDispatch()
	if _, err := oleutil.PutProperty(idispatch, "TargetPath", src); err != nil {
		return fmt.Errorf("PutProperty : %v", err)
	}
	if _, err := oleutil.CallMethod(idispatch, "Save"); err != nil {
		return fmt.Errorf("Save : %v", err)
	}

	return nil
}
