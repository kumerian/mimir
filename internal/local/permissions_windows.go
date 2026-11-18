package local

import (
	"fmt"
	"os"
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
	accessAllowedACEType     = 0
	accessDeniedACEType      = 1
	aclSizeInformationClass  = 2
)

type aclSizeInformation struct {
	aceCount      uint32
	aclBytesInUse uint32
	aclBytesFree  uint32
}

var (
	advapi32                            = syscall.NewLazyDLL("advapi32.dll")
	procConvertStringSecurityDescriptor = advapi32.NewProc("ConvertStringSecurityDescriptorToSecurityDescriptorW")
	procGetNamedSecurityInfo            = advapi32.NewProc("GetNamedSecurityInfoW")
	procGetSecurityDescriptorDACL       = advapi32.NewProc("GetSecurityDescriptorDacl")
	procGetACLInformation               = advapi32.NewProc("GetAclInformation")
	procGetACE                          = advapi32.NewProc("GetAce")
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

	var daclPresent, daclDefaulted uint32
	var dacl uintptr
	result, _, _ := procGetSecurityDescriptorDACL.Call(
		descriptor,
		uintptr(unsafe.Pointer(&daclPresent)),
		uintptr(unsafe.Pointer(&dacl)),
		uintptr(unsafe.Pointer(&daclDefaulted)),
	)
	if result == 0 || daclPresent == 0 || dacl == 0 {
		return false
	}

	var aclInfo aclSizeInformation
	result, _, _ = procGetACLInformation.Call(
		dacl,
		uintptr(unsafe.Pointer(&aclInfo)),
		unsafe.Sizeof(aclInfo),
		aclSizeInformationClass,
	)
	if result == 0 {
		return false
	}
	allowed := map[string]bool{
		userSID:        true,
		"S-1-5-18":     true,
		"S-1-5-32-544": true,
	}
	foundUserAccess := false
	for i := uint32(0); i < aclInfo.aceCount; i++ {
		var ace uintptr
		result, _, _ = procGetACE.Call(dacl, uintptr(i), uintptr(unsafe.Pointer(&ace)))
		if result == 0 || ace == 0 {
			return false
		}
		aceType := *(*byte)(unsafe.Pointer(ace))
		if aceType != accessAllowedACEType && aceType != accessDeniedACEType {
			return false
		}
		sid, err := (*syscall.SID)(unsafe.Pointer(ace + 8)).String()
		if err != nil || !allowed[sid] {
			return false
		}
		if aceType == accessAllowedACEType && sid == userSID {
			foundUserAccess = true
		}
	}
	return aclInfo.aceCount > 0 && foundUserAccess
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

func callError(err error) error {
	if err == syscall.Errno(0) {
		return syscall.EINVAL
	}
	return err
}
