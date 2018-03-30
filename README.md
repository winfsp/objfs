# objfs - Object Storage File System

Objfs is the "object storage file system". Objfs exposes objects from an object storage, such as a cloud drive, etc. as files in a file system that is fully integrated with the operating system. Programs that run on the operating system are able to access these files as if they are stored in a local "drive" (perhaps with some delay due to network operations).

- Supported operating systems: Windows, macOS, and Linux.
- Supported object storages: OneDrive

## How to use

Objfs is implemented as a command-line program that accepts commands such as `auth` and `mount`, but also shell-like commands, such as `ls`, `stat`, etc.

```
$ ./objfs help
usage: objfs [-options] command args...

commands:
  help
  auth
    	perform authentication/authorization
  mount
    	mount file system
  ls
    	list files
  stat
    	display file information
  mkdir
    	make directories
  rmdir
    	remove directories
  rm
    	remove files
  mv
    	move (rename) files
  get
    	get (download) files
  put
    	put (upload) files

options:
  -credentials path
    	auth credentials path (keyring:service/user or /file/path)
  -storage name
    	storage name to access (default "onedrive")
  -storage_uri uri
    	storage uri to access
  -v	verbose
```

### Auth

An object storage may need "auth" (authentication or authorization) before it can be mounted. The `objfs` program accepts credentials through the `-credentials path` option; the particulars credentials used depend on the auth mechanism that the object storage uses.

Some auth mechanisms (e.g. Oauth2) may support a mechanism where credentials that require user presence may be converted to long term credentials that do not require user presence. The `objfs` program supports such credentials through the `auth` command. The `auth` command will convert the credentials supplied by the `-credentials path` option to different (possibly long-term) credentials.

(Add full oauth2 flow here.)

### Mount

(Add `mount` info here.)

## How to build

Objfs is written in Go and uses [cgofuse](https://github.com/billziss-gh/cgofuse) to interface with the operating system. It requires the relevant FUSE drivers/libraries for each operating system.

Prerequisites:
- Windows: [WinFsp](https://github.com/billziss-gh/winfsp), gcc (e.g. from [Mingw-builds](http://mingw-w64.org/doku.php/download))
- macOS: [FUSE for macOS](https://osxfuse.github.io), [command line tools](https://developer.apple.com/library/content/technotes/tn2339/_index.html)
- Linux: libfuse-dev, gcc

To build ensure that you have updated all submodules and issue `make`:

```
$ git clone submodule update --recursive
$ make      # on windows you may have to update the PATH in make.cmd
```
