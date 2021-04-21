# SSVM-Go Fibonacci example

## Build

```bash
# In the current directory.
$ go get -u github.com/second-state/ssvm-go/ssvm
$ go build
```

## Run

```bash
$ ./print_fibonacci
```

The output will be as the following:

```
registered modules:  [host wasi_snapshot_preview1 wasm]
 --- Exported instances of the anonymous module
     --- Functions ( 1 ) :  [print_val_and_fib]
     --- Tables    ( 0 ) :  []
     --- Memories  ( 0 ) :  []
     --- Globals   ( 0 ) :  []
 --- Exported instances of module: host
     --- Functions ( 1 ) :  [print_val_and_res]
     --- Tables    ( 0 ) :  []
     --- Memories  ( 0 ) :  []
     --- Globals   ( 0 ) :  []
 --- Exported instances of module: wasi_snapshot_preview1
     --- Functions ( 45 ) :  [args_get args_sizes_get clock_res_get clock_time_get environ_get environ_sizes_get fd_advise fd_allocate fd_close fd_datasync fd_fdstat_get fd_fdstat_set_flags fd_fdstat_set_rights fd_filestat_get fd_filestat_set_size fd_filestat_set_times fd_pread fd_prestat_dir_name fd_prestat_get fd_pwrite fd_read fd_readdir fd_renumber fd_seek fd_sync fd_tell fd_write path_create_directory path_filestat_get path_filestat_set_times path_link path_open path_readlink path_remove_directory path_rename path_symlink path_unlink_file poll_oneoff proc_exit proc_raise random_get sched_yield sock_recv sock_send sock_shutdown]
     --- Tables    ( 0 ) :  []
     --- Memories  ( 0 ) :  []
     --- Globals   ( 0 ) :  []
 --- Exported instances of module: wasm
     --- Functions ( 1 ) :  [fib]
     --- Tables    ( 0 ) :  []
     --- Memories  ( 0 ) :  []
     --- Globals   ( 0 ) :  []
 ### Running print_val_and_fib with fib[ 20 ] ...
 [HostFunction] external value:  123456  , fibonacci number:  10946
 ### Running print_val_and_fib with fib[ 21 ] ...
 [HostFunction] external value:  876543210  , fibonacci number:  17711
 ### Running wasm::fib[ 22 ] ...
 Return value:  28657
```
