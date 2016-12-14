/*
compilecheck sets up tests to check the validity of generated code samples.

compilecheck does not perform any checking by itself. Instead, it sets up an environment for
testing and prints a command the user can run to perform the test.

Usage:
  compilecheck [-lib libDir] [-tst tstDir] [-pprof cpu.out] sampleDir...

For each argument in sampleDir, if the argument is a file, compilecheck sets up checks for the file.
If it is a directory, compilecheck sets up checks all files in the directory recursively.
Symlinks are not followed.

Compilecheck prints to the standard output shell command(s) that maybe run to perform the test.

Options
  -lib libDir
    It maybe necessary to download libraries to test against.
    If required, libraries are download into subdirectories of libDir.
    Libraries for different languages are downloaded into subdirectories named after the language,
    to prevent name collisions.

  -tst tstDir
    Files required for testing code samples are written into subdirectories of tstDir.
    Like -lib, files for different languages are written into different subdirectories.

  -pprof cpu.out
    If set, CPU profiling is written to cpu.out.
*/
package main
