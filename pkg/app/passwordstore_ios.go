//go:build ios

package app

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation

#include <stdlib.h>
#include <string.h>
#include <CoreFoundation/CoreFoundation.h>
#include <Security/Security.h>

static CFMutableDictionaryRef jk_query(const char *service, const char *account) {
	CFMutableDictionaryRef query = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFStringRef serviceRef = CFStringCreateWithCString(kCFAllocatorDefault, service, kCFStringEncodingUTF8);
	CFStringRef accountRef = CFStringCreateWithCString(kCFAllocatorDefault, account, kCFStringEncodingUTF8);
	CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword);
	CFDictionarySetValue(query, kSecAttrService, serviceRef);
	CFDictionarySetValue(query, kSecAttrAccount, accountRef);
	CFRelease(serviceRef);
	CFRelease(accountRef);
	return query;
}

static OSStatus jk_load(const char *service, const char *account, char **out, size_t *outLen) {
	CFMutableDictionaryRef query = jk_query(service, account);
	CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue);
	CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne);
	CFTypeRef result = NULL;
	OSStatus status = SecItemCopyMatching(query, &result);
	CFRelease(query);
	if (status != errSecSuccess) {
		return status;
	}
	CFDataRef data = (CFDataRef)result;
	CFIndex length = CFDataGetLength(data);
	*out = malloc(length > 0 ? (size_t)length : 1);
	memcpy(*out, CFDataGetBytePtr(data), (size_t)length);
	*outLen = (size_t)length;
	CFRelease(result);
	return errSecSuccess;
}

static OSStatus jk_save(const char *service, const char *account, const char *bytes, size_t length) {
	CFMutableDictionaryRef query = jk_query(service, account);
	CFDataRef value = CFDataCreate(kCFAllocatorDefault, (const UInt8 *)bytes, (CFIndex)length);
	CFMutableDictionaryRef update = CFDictionaryCreateMutable(kCFAllocatorDefault, 0,
		&kCFTypeDictionaryKeyCallBacks, &kCFTypeDictionaryValueCallBacks);
	CFDictionarySetValue(update, kSecValueData, value);
	OSStatus status = SecItemUpdate(query, update);
	if (status == errSecItemNotFound) {
		CFDictionarySetValue(query, kSecValueData, value);
		CFDictionarySetValue(query, kSecAttrAccessible, kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly);
		status = SecItemAdd(query, NULL);
	}
	CFRelease(update);
	CFRelease(value);
	CFRelease(query);
	return status;
}

static OSStatus jk_delete(const char *service, const char *account) {
	CFMutableDictionaryRef query = jk_query(service, account);
	OSStatus status = SecItemDelete(query);
	CFRelease(query);
	return status == errSecItemNotFound ? errSecSuccess : status;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// keychainService groups this app's device passwords in the Keychain.
const keychainService = "io.github.mwiget.jetkvm.device-password"

type keychainPasswordStore struct{}

func platformPasswordStore() devicePasswordStore {
	return keychainPasswordStore{}
}

func keychainStrings(baseURL string) (service, account *C.char, free func()) {
	service = C.CString(keychainService)
	account = C.CString(baseURL)
	return service, account, func() {
		C.free(unsafe.Pointer(service))
		C.free(unsafe.Pointer(account))
	}
}

func (keychainPasswordStore) Load(baseURL string) (string, bool) {
	service, account, free := keychainStrings(baseURL)
	defer free()
	var out *C.char
	var length C.size_t
	if C.jk_load(service, account, &out, &length) != 0 {
		return "", false
	}
	defer C.free(unsafe.Pointer(out))
	return C.GoStringN(out, C.int(length)), true
}

func (keychainPasswordStore) Save(baseURL, password string) error {
	service, account, free := keychainStrings(baseURL)
	defer free()
	bytes := C.CString(password)
	defer C.free(unsafe.Pointer(bytes))
	if status := C.jk_save(service, account, bytes, C.size_t(len(password))); status != 0 {
		return fmt.Errorf("keychain: save: OSStatus %d", status)
	}
	return nil
}

func (keychainPasswordStore) Delete(baseURL string) error {
	service, account, free := keychainStrings(baseURL)
	defer free()
	if status := C.jk_delete(service, account); status != 0 {
		return fmt.Errorf("keychain: delete: OSStatus %d", status)
	}
	return nil
}
