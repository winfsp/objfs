# objfs - Object Storage File System

The [objfs](https://github.com/billziss-gh/objfs) repository and its companion repository [objfs.pkg](https://github.com/billziss-gh/objfs.pkg) contain the implementation of objfs, the "object storage file system".

Objfs exposes objects from an object storage, such as a cloud drive, etc. as files in a file system that is fully integrated with the operating system. Programs that run on the operating system are able to access these files as if they are stored in a local "drive" (perhaps with some delay due to network operations).

- Supported operating systems: Windows, macOS, and Linux.
- Supported object storages: OneDrive

## How to use

Objfs is implemented as a command-line program that accepts commands such as `auth` and `mount`, but also shell-like commands, such as `ls`, `stat`, etc.

```
$ ./objfs help
usage: objfs [-options] command args...

commands:
  version
    	get current version information
  config
    	get or set configuration options
  auth
    	perform authentication/authorization
  mount
    	mount file system
  statfs
    	get storage information
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
  cache-pending
    	list pending cache files
  cache-reset
    	reset cache (upload and evict files)

options:
  -accept-tls-cert
    	accept any TLS certificate presented by the server (insecure)
  -auth name
    	auth name to use
  -config path
    	path to configuration file
  -credentials path
    	auth credentials path (keyring:service/user or /file/path)
  -datadir path
    	path to supporting data and caches
  -storage name
    	storage name to access (default "onedrive")
  -storage-uri uri
    	storage uri to access
  -v	verbose
```

### Default Storage

Objfs uses defaults to simplify command line invocation. In the default build of objfs, the default storage is `onedrive`.

### Auth

Objfs supports multiple "auth" (authentication or authorization) mechanisms through the `-credentials path` option and the `auth` command.

In general before an object storage service can be used it requires auth. The specific auth mechanism used depends on the service and it ranges from no auth, to username/password, to Oauth2, etc. Auth mechanisms require credentials, which can be supplied using the `-credentials path` option.

In some cases the object storage service cannot readily accept the supplied credentials, they must be converted to other credentials first. As an authentication example, a particular service may require username/password credentials to be converted to some form of service-level token before they can be used. As an authorization example Oauth2 requires application-level credentials together with user consent to form a service-level token that can be used to access the service.

The `auth` command can be used for this purpose. It takes user-level or application-level credentials and converts them to service-level credentials.

Credentials can be stored in the local file system or the system keyring. The syntax `/file/path` is used to name credentials stored in the file system. The syntax `keyring:service/user` is used to name credentials stored in the system keyring.

#### Example - Oauth2 Flow

- Prepare the Oauth2 `client_secret` credentials in a file or the system keyring:
    ```
    client_id="XXXXXXXX"
    client_secret="XXXXXXXX"
    redirect_uri="http://localhost:xxxxx"
    scope="files.readwrite.all offline_access"
    ```
- Issue the command:
    ```
    $ ./objfs -credentials=CLIENT_SECRET_PATH auth TOKEN_PATH
    ```
- This will launch your browser and ask for authorization. If the access is authorized the Oauth2 `access_token` and `refresh_token` will be stored in the specified path.
- The object storage can now be mounted using the command:
    ```
    $ ./objfs -credentials=TOKEN_PATH mount MOUNTPOINT
    ```

### Mount

The objfs `mount` command is used to mount an object storage as a file system on a mountpoint. On Windows the mount point must be a non-existing drive or directory; it is recommended that an object storage is only mounted as a drive when the object storage is case-sensitive. On macOS and Linux the mount point must be an existing directory.

To mount on Windows:

```
> objfs -credentials=TOKEN_PATH mount -o uid=-1,gid=-1 mount X:
```

To mount on macOS and Linux:

```
$ ./objfs -credentials=TOKEN_PATH mount MOUNTPOINT
```

Objfs uses a local file cache to speed up file system operations. This caches files locally when they are first opened; subsequent I/O operations will be performed against the local file and are therefore fast. Modified files will be uploaded to the object storage when they are closed. File system operations such as creating and deleting files and listing directories are sent directly to the object storage and are therefore slow (although some of their results are cached).

The Objfs cache was inspired by an early version of the Andrew File System (AFS). For more information see this [paper](http://pages.cs.wisc.edu/~remzi/OSTEP/dist-afs.pdf).

### Diagnostics

Objfs includes a tracing facility that can be used to troubleshoot problems, to gain insights into its internal workings, etc. This facility is enabled when the `-v` option is used.

The environment variable `GOLIB_TRACE` controls which traces are enabled. This variable accepts a comma separated list of file-style patterns containing wildcards such as `*` and `?`.

```
$ export GOLIB_TRACE=pattern1,...,patternN
```

Examples:

```
$ export GOLIB_TRACE=github.com/billziss-gh/objfs/fs.*      # file system traces
$ export GOLIB_TRACE=github.com/billziss-gh/objfs/objio.*   # object storage traces
$ export GOLIB_TRACE=github.com/billziss-gh/objfs/fs.*,github.com/billziss-gh/objfs/objio.*
$ ./objfs -v -credentials=TOKEN_PATH mount MOUNTPOINT
```

## How to build

Objfs is written in Go and uses [cgofuse](https://github.com/billziss-gh/cgofuse) to interface with the operating system. It requires the relevant FUSE drivers/libraries for each operating system.

Prerequisites:
- Windows: [WinFsp](https://github.com/billziss-gh/winfsp), gcc (e.g. from [Mingw-builds](http://mingw-w64.org/doku.php/download))
- macOS: [FUSE for macOS](https://osxfuse.github.io), [command line tools](https://developer.apple.com/library/content/technotes/tn2339/_index.html)
- Linux: libfuse-dev, gcc

To build the following is usually sufficient:

```
$ go get -d github.com/billziss-gh/objfs
$ make      # on windows you may have to update the PATH in make.cmd
```

This will include all supported storages. Objfs storage and auth mechanisms are maintained in the separate repository objfs.pkg. You can customize the supported storages for licensing or other reasons by modifying the [`Makefile Packages`](Makefile) variable.

## License

The objfs and objfs.pkg repositories are available under the [AGPLv3](License.txt) license.

The project has the following dependencies and their licensing:

- [cgofuse](https://github.com/billziss-gh/cgofuse) - Cross-platform FUSE library for Go.
    - License: MIT
- [boltdb](https://github.com/boltdb/bolt) - An embedded key/value database for Go.
    - License: MIT
- [golib](https://github.com/billziss-gh/golib) - Collection of Go libraries.
    - License: MIT
- [oauth2-helper](https://github.com/billziss-gh/oauth2-helper) - OAuth 2.0 for Native Apps
    - License: MIT
- [WinFsp](https://github.com/billziss-gh/winfsp) - Windows File System Proxy - FUSE for Windows.
    - License: GPLv3 w/ FLOSS exception
- [FUSE for macOS](https://osxfuse.github.io) - File system integration made easy.
    - License: BSD-style
- [libfuse](https://github.com/libfuse/libfuse) - The reference implementation of the Linux FUSE (Filesystem in Userspace) interface.
    - License: LGPLv2.1
