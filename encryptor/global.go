package encryptor

import "runtime"

const encExt = ".ji"
const nonceSize = 12
const gcmTagSize = 16
const chunkSize = 1024 * 1024 * 1
const enc_chunkSize = chunkSize + nonceSize + gcmTagSize

var DefaultGoRoutines = 100
var DefaultMaxFiles = 10

var MaxCPUs = runtime.NumCPU()
