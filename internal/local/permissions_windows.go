package local

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

const (
	seFileObject             = 1
	daclSecurityInformation  = 0x00000004
	protectedDACLInformation = 0x80000000
	sddlRevision             = 1
	fileShareRead            = 0x00000001
	fileShareWrite           = 0x00000002
	fileShareDelete          = 0x00000004
	fileFlagBackupSemantics  = 0x02000000
)

var (
	advapi32                              = syscall.NewLazyDLL("advapi32.dll")
	procConvertStringSecurityDescriptor   = advapi32.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
	procConvertSecurityDescriptorToString = advapi32.NewProc("ConvertSecurityDescriptorToStringSecurityDescriptorW")
	procGetNamedSecurityInfo              = advapi32.NewProc("GetNamedSecurityInfoW")
)

func createPrivateDirectory(path string) error {
	userSID, err := currentUserSID()
	if err != nil {
		return err
	}
	descriptor, err := securityDescriptor(fmt.Sprintf("D:P(A;OICI;FA;;;%s)(A;OICI;FA;;;SY)(A;OICI;FA;;;BA)", userSID))
	if err != nil {
		return err
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))

	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	attributes := syscall.SecurityAttributes{
		Length:             uint32(unsafe.Sizeof(syscall.SecurityAttributes{})),
		SecurityDescriptor: descriptor,
	}
	return syscall.CreateDirectory(pathPointer, &attributes)
}

func isPrivateDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir() && isPrivatePath(path)
}

func isPrivateFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && isPrivatePath(path)
}

func isPrivatePath(path string) bool {
	userSID, err := currentUserSID()
	if err != nil {
		return false
	}
	descriptor, err := namedSecurityDescriptor(path)
	if err != nil {
		return false
	}
	defer syscall.LocalFree(syscall.Handle(descriptor))

	sddl, err := securityDescriptorString(descriptor)
	if err != nil {
		return false
	}
	daclStart := strings.Index(sddl, "D:")
	if daclStart < 0 {
		return false
	}
	dacl := sddl[daclStart+2:]
	allowed := map[string]bool{
		userSID:        true,
		"SY":           true,
		"BA":           true,
		"S-1-5-18":     true,
		"S-1-5-32-544": true,
	}
	foundUserAccess := false
	aceCount := 0
	for len(dacl) > 0 {
		start := strings.IndexByte(dacl, '(')
		if start < 0 {
			break
		}
		end := strings.IndexByte(dacl[start+1:], ')')
		if end < 0 {
			return false
		}
		ace := strings.Split(dacl[start+1:start+1+end], ";")
		if len(ace) != 6 || (ace[0] != "A" && ace[0] != "D") || !allowed[ace[5]] {
			return false
		}
		if ace[0] == "A" && ace[5] == userSID {
			foundUserAccess = true
		}
		aceCount++
		dacl = dacl[start+end+2:]
	}
	return aceCount > 0 && foundUserAccess
}

func syncDirectory(path string) error {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	directory, err := syscall.CreateFile(
		pathPointer,
		syscall.GENERIC_WRITE,
		fileShareRead|fileShareWrite|fileShareDelete,
		nil,
		syscall.OPEN_EXISTING,
		fileFlagBackupSemantics,
		0,
	)
	if err != nil {
		return err
	}
	syncErr := syscall.FlushFileBuffers(directory)
	closeErr := syscall.CloseHandle(directory)
	if syncErr != nil {
		return fmt.Errorf("sync directory: %w", syncErr)
	}
	if closeErr != nil {
		return closeErr
	}
	return nil
}

func currentUserSID() (string, error) {
	token, err := syscall.OpenCurrentProcessToken()
	if err != nil {
		return "", err
	}
	defer token.Close()
	user, err := token.GetTokenUser()
	if err != nil {
		return "", err
	}
	return user.User.Sid.String()
}

func securityDescriptor(sddl string) (uintptr, error) {
	sddlPointer, err := syscall.UTF16PtrFromString(sddl)
	if err != nil {
		return 0, err
	}
	var descriptor uintptr
	var descriptorSize uint32
	result, _, callErr := procConvertStringSecurityDescriptor.Call(
		uintptr(unsafe.Pointer(sddlPointer)),
		sddlRevision,
		uintptr(unsafe.Pointer(&descriptor)),
		uintptr(unsafe.Pointer(&descriptorSize)),
	)
	if result == 0 {
		return 0, callError(callErr)
	}
	return descriptor, nil
}

func namedSecurityDescriptor(path string) (uintptr, error) {
	pathPointer, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var descriptor uintptr
	result, _, _ := procGetNamedSecurityInfo.Call(
		uintptr(unsafe.Pointer(pathPointer)),
		seFileObject,
		daclSecurityInformation,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&descriptor)),
	)
	if result != 0 {
		return 0, syscall.Errno(result)
	}
	return descriptor, nil
}

func securityDescriptorString(descriptor uintptr) (string, error) {
	var sddlPointer *uint16
	var sddlLength uint32
	result, _, callErr := procConvertSecurityDescriptorToString.Call(
		descriptor,
		sddlRevision,
		daclSecurityInformation,
		uintptr(unsafe.Pointer(&sddlPointer)),
		uintptr(unsafe.Pointer(&sddlLength)),
	)
	if result == 0 {
		return "", callError(callErr)
	}
	defer syscall.LocalFree(syscall.Handle(uintptr(unsafe.Pointer(sddlPointer))))
	return syscall.UTF16ToString(unsafe.Slice(sddlPointer, sddlLength)), nil
}

func callError(err error) error {
	if err == syscall.Errno(0) {
		return syscall.EINVAL
	}
	return err
}
