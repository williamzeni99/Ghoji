# ghoji 

ghoji (Go-Hoji, from japanese Hoji means retention) is a CLI tool for encrypting files with GO. It implements AES256 with GCM, with 1MB chunk size. 
Each chunk is encrypted in a goroutine, so it is fully parallelized. During the encryption you can set the number of physical cores you want to use
and the number of max goroutines you want to run in parallel. No checks are done on the status of the ram memory usage, so an high number of goroutines
will cause a crash. Right now you can just encrypt a file per time (also big ones, I tested 10GB file on a 8GB ram and a Intel(R) Core(TM) i5-8300H CPU @ 2.30GHz and it took 15seconds), but improvments are going to be implemented.


UPDATE: 
Now you can encrypt a directory. It will encrypt each file contained in the directory. You can set up the number of files to encrypt in parallel. Next update will be making a single encrypted file. 
