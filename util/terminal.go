package util

import "os"

func SetKeypadToNumericMode() {
	sendEscapeSequence(0x1B, 0x3E)
}

func sendEscapeSequence(sequence ...byte) {
	os.Stdout.Write(sequence)
}
