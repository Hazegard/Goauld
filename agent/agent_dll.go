//go:build dll

package main

import (
	"C"
	"os"
	"unsafe"
)

//export Main2
func Main2() {
	main()
}

//export Main
func Main(argc C.int, argv **C.char) {
	if int(argc) == 0 {
		main()
	}
	goArgs := make([]string, int(argc))
	argPtrs := (*[1 << 28]*C.char)(unsafe.Pointer(argv))[:argc:argc]

	for i := 0; i < int(argc); i++ {
		goArgs[i] = C.GoString(argPtrs[i])
	}
	os.Args = append(os.Args, goArgs...)
	main()
}
