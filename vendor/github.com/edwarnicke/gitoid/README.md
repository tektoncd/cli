gitoid provides a simple library to compute gitoids (git object ids)

## Creating GitOIDs

### Default Usage
By default it produces gitoids for git object type blob using sha1:

```go
var reader os.Reader
gitoidHash, err := gitoid.New(reader)
fmt.Println(gitoidHash)
// Output: 261eeb9e9f8b2b4b0d119366dda99c6fd7d35c64
fmt.Println(gitoidHash.URI())
// Output: gitoid:blob:sha1:261eeb9e9f8b2b4b0d119366dda99c6fd7d35c64
```

### GitOid from string or []byte

It's simple to compute the gitoid from a string or []byte by using bytes.NewBuffer:

```go
input := []byte("example")
gitoidHash, _ := gitoid.New(bytes.NewBuffer(input))
fmt.Println(gitoidHash)
// Output: 96236f8158b12701d5e75c14fb876c4a0f31b963
fmt.Println(gitoidHash.URI())
// Output: gitoid:blob:sha1:96236f8158b12701d5e75c14fb876c4a0f31b963
```

### GitOID from URIs

GitOIDs can be represented as a [gitoid uri](https://www.iana.org/assignments/uri-schemes/prov/gitoid).

```go
gitoidHash, _ := gitoid.FromURI("gitoid:blob:sha1:96236f8158b12701d5e75c14fb876c4a0f31b96")
fmt.Println(gitoidHash)
// Output: 96236f8158b12701d5e75c14fb876c4a0f31b963
fmt.Println(gitoidHash.URI())
// Output: gitoid:blob:sha1:96236f8158b12701d5e75c14fb876c4a0f31b963
```

## Variations on GitOIDs

### SHA256 gitoids

Git defaults to computing gitoids with sha1.  Git also supports sha256 gitoids.  Sha256 gitoids are supported using
an Option:

```go
var reader os.Reader
gitoidHash, err := gitoid.New(reader, gitoid.WithSha256())
fmt.Println(gitoidHash)
// Output: ed43975fbdc3084195eb94723b5f6df44eeeed1cdda7db0c7121edf5d84569ab
fmt.Println(gitoidHash.URI())
// Output: gitoid:blob:sha256:ed43975fbdc3084195eb94723b5f6df44eeeed1cdda7db0c7121edf5d84569ab
```

### Other git object types

git has four object types: blob, tree, commit, tag.  By default gitoid using object type blob.
You may optionally specify another object type using an Option:

```go
var reader os.Reader
gitoidHash, err := gitoid.New(reader, gitoid.WithGitObjectType(gitoid.COMMIT))
```

### Assert ContentLength

git object ids consist of hash over a header followed by the file contents.  The header contains the length of the file
contents.  By default, gitoid simply copies the reader into a buffer to establish its contentLength to compute the header.

If you wish to assert the contentLength yourself, you may do so with an Option: 

```go
var reader os.Reader
var contentLength int64
gitoidHash, _ := gitoid.New(reader, gitoid.WithContentLength(contentLength))
fmt.Println(gitoidHash)
// Output: 261eeb9e9f8b2b4b0d119366dda99c6fd7d35c64
```

gitoid will read the first contentLength bytes from the provided reader.  If the reader is unable to provide
contentLength bytes a wrapper error around io.ErrUnexpectedEOF will be returned from gitoid.New

## Using GitOIDs

### Match contents to a GitOID

```go
var reader io.Reader
var gitoidHash *gitoid.GitOID
if gitoidHash.Match(reader) {
	fmt.Println("matched")
}
```

### Find files that match GitOID

```go
var path1 fs.FS = os.DirFS("./relative/path")
var path2 fs.FS = os.DirFS("/absolute/path")
var gitoidHash *gitoid.GitOID

// Find a file in path1 and path2 that matches gitoidHash
file,_ := gitoidHash.Find(path1, path2)

// Find all files in path1 and path2 that matches gitoidHash
files, := gitoidHash.FindAll(path1, path2)
```

