#include "shims.h"
#include "_cgo_export.h"
#include <errno.h>
#include <fcntl.h>
#include <stdlib.h>

#if defined(_WIN32)
#include <windows.h>
#else
#include <unistd.h>
#endif

// ---- opaque tokens ---------------------------------------------------------

void *wasmedgego_tokenCreate(void) { return malloc(1); }

void wasmedgego_tokenDelete(void *Token) { free(Token); }

// ---- WASI discard stdio ----------------------------------------------------

#if defined(_WIN32)

// The official WasmEdge DLL resolves its fd functions from this UCRT API set,
// while cgo's MinGW compiler links ordinary calls against msvcrt.dll. Those
// runtimes have separate fd tables, so passing an fd opened by the latter to
// WasmEdge would be invalid even when the integer happens to name a live slot.
// Resolve the same UCRT entry points as WasmEdge and retain the module for the
// process lifetime so the cached function pointers cannot dangle.
typedef int(__cdecl *wasmedgego_open_osfhandle_t)(intptr_t, int);
typedef int(__cdecl *wasmedgego_ucrt_close_t)(int);
typedef intptr_t(__cdecl *wasmedgego_get_osfhandle_t)(int);

struct wasmedgego_ucrtStdioAPI {
  HMODULE Module;
  wasmedgego_open_osfhandle_t OpenOSFHandle;
  wasmedgego_ucrt_close_t Close;
  wasmedgego_get_osfhandle_t GetOSFHandle;
};

static INIT_ONCE wasmedgego_ucrtStdioOnce = INIT_ONCE_STATIC_INIT;
static struct wasmedgego_ucrtStdioAPI wasmedgego_ucrtStdio;

static BOOL CALLBACK wasmedgego_initUCRTStdio(PINIT_ONCE Once, PVOID Parameter,
                                              PVOID *Context) {
  (void)Once;
  (void)Parameter;
  (void)Context;
  HMODULE Module = LoadLibraryW(L"api-ms-win-crt-stdio-l1-1-0.dll");
  if (!Module) {
    return TRUE;
  }

  wasmedgego_ucrtStdio.Module = Module;
  wasmedgego_ucrtStdio.OpenOSFHandle =
      (wasmedgego_open_osfhandle_t)GetProcAddress(Module, "_open_osfhandle");
  wasmedgego_ucrtStdio.Close =
      (wasmedgego_ucrt_close_t)GetProcAddress(Module, "_close");
  wasmedgego_ucrtStdio.GetOSFHandle =
      (wasmedgego_get_osfhandle_t)GetProcAddress(Module, "_get_osfhandle");
  if (!wasmedgego_ucrtStdio.OpenOSFHandle || !wasmedgego_ucrtStdio.Close ||
      !wasmedgego_ucrtStdio.GetOSFHandle) {
    FreeLibrary(Module);
    wasmedgego_ucrtStdio.Module = NULL;
    wasmedgego_ucrtStdio.OpenOSFHandle = NULL;
    wasmedgego_ucrtStdio.Close = NULL;
    wasmedgego_ucrtStdio.GetOSFHandle = NULL;
  }
  return TRUE;
}

static int wasmedgego_requireUCRTStdio(void) {
  if (!InitOnceExecuteOnce(&wasmedgego_ucrtStdioOnce, wasmedgego_initUCRTStdio,
                           NULL, NULL) ||
      !wasmedgego_ucrtStdio.Module) {
    errno = ENOSYS;
    return -1;
  }
  return 0;
}

static int wasmedgego_openNull(int Write) {
  if (wasmedgego_requireUCRTStdio() != 0) {
    return -1;
  }
  HANDLE Handle =
      CreateFileW(L"NUL", Write ? GENERIC_WRITE : GENERIC_READ,
                  FILE_SHARE_READ | FILE_SHARE_WRITE | FILE_SHARE_DELETE, NULL,
                  OPEN_EXISTING, FILE_ATTRIBUTE_NORMAL, NULL);
  if (Handle == INVALID_HANDLE_VALUE) {
    errno = EIO;
    return -1;
  }
  const int Flags = (Write ? _O_WRONLY : _O_RDONLY) | _O_BINARY | _O_NOINHERIT;
  const int Fd = wasmedgego_ucrtStdio.OpenOSFHandle((intptr_t)Handle, Flags);
  if (Fd < 0) {
    CloseHandle(Handle);
    errno = EMFILE;
  }
  return Fd;
}

static int wasmedgego_closeFd(int Fd) {
  if (wasmedgego_requireUCRTStdio() != 0) {
    return -1;
  }
  return wasmedgego_ucrtStdio.Close(Fd);
}

static intptr_t wasmedgego_nativeHandler(int Fd) {
  if (wasmedgego_requireUCRTStdio() != 0) {
    return -1;
  }
  return wasmedgego_ucrtStdio.GetOSFHandle(Fd);
}

#else

static int wasmedgego_openNull(int Write) {
  int Flags = Write ? O_WRONLY : O_RDONLY;
#ifdef O_CLOEXEC
  Flags |= O_CLOEXEC;
#endif
  return open("/dev/null", Flags);
}

static int wasmedgego_closeFd(int Fd) { return close(Fd); }

static intptr_t wasmedgego_nativeHandler(int Fd) {
  return fcntl(Fd, F_GETFD) == -1 ? -1 : (intptr_t)Fd;
}

#endif

int wasmedgego_wasiOpenNullFds(int32_t *StdInFd, int32_t *StdOutFd,
                               int32_t *StdErrFd, uint64_t *StdInHandler,
                               uint64_t *StdOutHandler,
                               uint64_t *StdErrHandler) {
  if (!StdInFd || !StdOutFd || !StdErrFd || !StdInHandler || !StdOutHandler ||
      !StdErrHandler) {
    errno = EINVAL;
    return -1;
  }

  int Fds[3] = {-1, -1, -1};
  Fds[0] = wasmedgego_openNull(0);
  if (Fds[0] >= 0) {
    Fds[1] = wasmedgego_openNull(1);
  }
  if (Fds[1] >= 0) {
    Fds[2] = wasmedgego_openNull(1);
  }
  intptr_t Handlers[3] = {-1, -1, -1};
  if (Fds[0] >= 0 && Fds[1] >= 0 && Fds[2] >= 0) {
    for (size_t I = 0; I < 3; I++) {
      Handlers[I] = wasmedgego_nativeHandler(Fds[I]);
    }
    if (Handlers[0] == -1 || Handlers[1] == -1 || Handlers[2] == -1) {
      errno = EBADF;
    }
  }
  if (Fds[0] < 0 || Fds[1] < 0 || Fds[2] < 0 || Handlers[0] == -1 ||
      Handlers[1] == -1 || Handlers[2] == -1) {
    const int SavedErrno = errno;
    for (size_t I = 0; I < 3; I++) {
      if (Fds[I] >= 0) {
        (void)wasmedgego_closeFd(Fds[I]);
      }
    }
    errno = SavedErrno;
    return -1;
  }

  *StdInFd = (int32_t)Fds[0];
  *StdOutFd = (int32_t)Fds[1];
  *StdErrFd = (int32_t)Fds[2];
  *StdInHandler = (uint64_t)(uintptr_t)Handlers[0];
  *StdOutHandler = (uint64_t)(uintptr_t)Handlers[1];
  *StdErrHandler = (uint64_t)(uintptr_t)Handlers[2];
  return 0;
}

void wasmedgego_wasiCloseFds(int32_t StdInFd, int32_t StdOutFd,
                             int32_t StdErrFd) {
  const int32_t Fds[3] = {StdInFd, StdOutFd, StdErrFd};
  for (size_t I = 0; I < 3; I++) {
    if (Fds[I] >= 0) {
      (void)wasmedgego_closeFd((int)Fds[I]);
    }
  }
}

int wasmedgego_wasiHandlerIsOpen(uint64_t Handler) {
#if defined(_WIN32)
  DWORD Flags;
  return GetHandleInformation((HANDLE)(uintptr_t)Handler, &Flags) != 0;
#else
  return Handler <= INT32_MAX && fcntl((int)Handler, F_GETFD) != -1;
#endif
}

// ---- v128 ------------------------------------------------------------------

WasmEdge_Value wasmedgego_valueGenV128(uint64_t Low, uint64_t High) {
#if defined(__x86_64__) || defined(__aarch64__) ||                             \
    (defined(__riscv) && __riscv_xlen == 64) || defined(__s390x__)
  uint128_t Bits = ((uint128_t)High << 64) | (uint128_t)Low;
  return WasmEdge_ValueGenV128((int128_t)Bits);
#else
  int128_t Bits = {.Low = Low, .High = (int64_t)High};
  return WasmEdge_ValueGenV128(Bits);
#endif
}

void wasmedgego_valueGetV128(WasmEdge_Value Value, uint64_t *Low,
                             uint64_t *High) {
  int128_t Signed = WasmEdge_ValueGetV128(Value);
#if defined(__x86_64__) || defined(__aarch64__) ||                             \
    (defined(__riscv) && __riscv_xlen == 64) || defined(__s390x__)
  uint128_t Bits = (uint128_t)Signed;
  *Low = (uint64_t)Bits;
  *High = (uint64_t)(Bits >> 64);
#else
  *Low = Signed.Low;
  *High = (uint64_t)Signed.High;
#endif
}

// ---- logging ---------------------------------------------------------------

static void wasmedgego_logShim(const WasmEdge_LogMessage *Msg) {
  // Cast away const: the cgo-generated prototype for the exported Go
  // function has no const qualifier. The Go side treats it as read-only.
  wasmedgego_logCallback((WasmEdge_LogMessage *)Msg);
}

void wasmedgego_setLogCallback(int enable) {
  WasmEdge_LogSetCallback(enable ? wasmedgego_logShim : NULL);
}

// ---- host functions --------------------------------------------------------

static WasmEdge_Result
wasmedgego_wrapShim(void *This, void *Data,
                    const WasmEdge_CallingFrameContext *CallFrame,
                    const WasmEdge_Value *Params, const uint32_t ParamLen,
                    WasmEdge_Value *Returns, const uint32_t ReturnLen) {
  (void)Data;
  return wasmedgego_hostFuncInvoke(
      This, (WasmEdge_CallingFrameContext *)CallFrame, (WasmEdge_Value *)Params,
      ParamLen, Returns, ReturnLen);
}

WasmEdge_FunctionInstanceContext *
wasmedgego_functionCreate(const WasmEdge_FunctionTypeContext *Type, void *Token,
                          uint64_t Cost) {
  return WasmEdge_FunctionInstanceCreateBinding(Type, wasmedgego_wrapShim,
                                                Token, NULL, Cost);
}

// ---- module host data ------------------------------------------------------

static void wasmedgego_moduleDataFinalizeShim(void *Data) {
  wasmedgego_moduleDataFinalize(Data);
}

WasmEdge_ModuleInstanceContext *
wasmedgego_moduleCreateWithData(WasmEdge_String Name, void *Token) {
  return WasmEdge_ModuleInstanceCreateWithData(
      Name, Token, wasmedgego_moduleDataFinalizeShim);
}
