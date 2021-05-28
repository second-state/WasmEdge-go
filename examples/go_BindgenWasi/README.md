# WasmEdge-Go Wasm-Bindgen WASI example

This example is a rust to WASM with `wasm-bindgen`. This example is modified from the [nodejs WASM example](https://github.com/second-state/wasm-learning/tree/master/nodejs/wasi).

## Build

```bash
# In the current directory.
$ go get -u github.com/second-state/WasmEdge-go
$ go build
```

## (Optional) Build the example WASM from rust

The pre-built WASM from rust is provided as "rust_bindgen_wasi_lib_bg.wasm".
The pre-built compiled-WASM from rust is provided as "rust_bindgen_wasi_lib_bg.so".

For building the WASM from the rust source, the following steps are required:

* Install the [rustc and cargo](https://www.rust-lang.org/tools/install).
* Set the default `rustup` version to `1.50.0` or lower.
  * `$ rustup default 1.50.0`
* Install the [rustwasmc](https://github.com/second-state/rustwasmc)
  * `$ curl https://raw.githubusercontent.com/second-state/rustwasmc/master/installer/init.sh -sSf | sh`

```bash
$ cd rust_bindgen_wasi
$ rustwasmc build --enable-aot
# The output WASM will be `pkg/rust_bindgen_wasi_lib_bg.wasm`.
# The output compiled-WASM will be `pkg/rust_bindgen_wasi_lib_bg.so`.
```

## Run

```bash
# For the interpreter mode
$ ./bindgen_wasi rust_bindgen_wasi_lib_bg.wasm
# For the AOT mode
$ ./bindgen_wasi rust_bindgen_wasi_lib_bg.so
```

The standard output of this example will be the following:

```bash
Run bindgen -- get_random_i32: 735310784
Run bindgen -- get_random_bytes: [70 191 126 79 80 236 193 206 87 113 5 143 82 61 35 57 205 64 139 169 54 116 167 61 221 244 220 134 172 30 19 94 87 36 153 236 71 204 49 68 50 60 192 1 139 191 183 225 110 81 16 240 51 195 254 206 51 28 209 208 111 80 60 70 10 36 103 247 159 201 246 136 19 108 201 189 185 169 227 225 135 47 39 232 140 189 156 47 242 57 149 6 199 39 244 66 76 237 7 47 210 69 224 39 161 85 102 35 138 215 4 231 115 164 105 127 83 240 21 86 185 182 48 217 6 90 22 142]
Printed from wasi: hello!!!!
Run bindgen -- echo: hello!!!!
The env vars are as follows.
HOSTNAME: e61a614e7306
PWD: /root/go_BindgenWasi
CXX: /usr/bin/clang++
HOME: /root
LS_COLORS: rs=0:di=01;34:ln=01;36:mh=00:pi=40;33:so=01;35:do=01;35:bd=40;33;01:cd=40;33;01:or=40;31;01:mi=00:su=37;41:sg=30;43:ca=30;41:tw=30;42:ow=34;42:st=37;44:ex=01;32:*.tar=01;31:*.tgz=01;31:*.arc=01;31:*.arj=01;31:*.taz=01;31:*.lha=01;31:*.lz4=01;31:*.lzh=01;31:*.lzma=01;31:*.tlz=01;31:*.txz=01;31:*.tzo=01;31:*.t7z=01;31:*.zip=01;31:*.z=01;31:*.dz=01;31:*.gz=01;31:*.lrz=01;31:*.lz=01;31:*.lzo=01;31:*.xz=01;31:*.zst=01;31:*.tzst=01;31:*.bz2=01;31:*.bz=01;31:*.tbz=01;31:*.tbz2=01;31:*.tz=01;31:*.deb=01;31:*.rpm=01;31:*.jar=01;31:*.war=01;31:*.ear=01;31:*.sar=01;31:*.rar=01;31:*.alz=01;31:*.ace=01;31:*.zoo=01;31:*.cpio=01;31:*.7z=01;31:*.rz=01;31:*.cab=01;31:*.wim=01;31:*.swm=01;31:*.dwm=01;31:*.esd=01;31:*.jpg=01;35:*.jpeg=01;35:*.mjpg=01;35:*.mjpeg=01;35:*.gif=01;35:*.bmp=01;35:*.pbm=01;35:*.pgm=01;35:*.ppm=01;35:*.tga=01;35:*.xbm=01;35:*.xpm=01;35:*.tif=01;35:*.tiff=01;35:*.png=01;35:*.svg=01;35:*.svgz=01;35:*.mng=01;35:*.pcx=01;35:*.mov=01;35:*.mpg=01;35:*.mpeg=01;35:*.m2v=01;35:*.mkv=01;35:*.webm=01;35:*.ogm=01;35:*.mp4=01;35:*.m4v=01;35:*.mp4v=01;35:*.vob=01;35:*.qt=01;35:*.nuv=01;35:*.wmv=01;35:*.asf=01;35:*.rm=01;35:*.rmvb=01;35:*.flc=01;35:*.avi=01;35:*.fli=01;35:*.flv=01;35:*.gl=01;35:*.dl=01;35:*.xcf=01;35:*.xwd=01;35:*.yuv=01;35:*.cgm=01;35:*.emf=01;35:*.ogv=01;35:*.ogx=01;35:*.aac=00;36:*.au=00;36:*.flac=00;36:*.m4a=00;36:*.mid=00;36:*.midi=00;36:*.mka=00;36:*.mp3=00;36:*.mpc=00;36:*.ogg=00;36:*.ra=00;36:*.wav=00;36:*.oga=00;36:*.opus=00;36:*.spx=00;36:*.xspf=00;36:
LESSCLOSE: /usr/bin/lesspipe %s %s
TERM: xterm
LESSOPEN: | /usr/bin/lesspipe %s
SHLVL: 1
PATH: /usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin
CC: /usr/bin/clang
DEBIAN_FRONTEND: noninteractive
OLDPWD: /root/go-proj
_: ./bindgen_wasi
The args are as follows.
rust_bindgen_wasi_lib_bg.wasm
Run bindgen -- print_env
Run bindgen -- create_file: test.txt
Run bindgen -- read_file: TEST MESSAGES----!@#@%@%$#!@#
Run bindgen -- del_file: test.txt
```