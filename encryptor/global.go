package encryptor

import "runtime"

const encExt = ".war"
const nonceSize = 12
const gcmTagSize = 16
const chunkSize = 1024 * 1024 * 1
const enc_chunkSize = chunkSize + nonceSize + gcmTagSize

const defaultGoRoutines = 1000

var maxCPUs = runtime.NumCPU()
