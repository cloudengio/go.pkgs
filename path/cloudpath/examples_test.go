package cloudpath_test

import (
	"fmt"

	"cloudeng.io/path/cloudpath"
)

func ExampleScheme() {
	for _, example := range []string{
		"s3://my-bucket/object",
		"https://storage.cloud.google.com/bucket/obj",
		"gs://my-bucket",
		`c:\root\file`,
	} {
		scheme := cloudpath.Scheme(example)
		local := cloudpath.IsLocal(example)
		host := cloudpath.Host(example)
		volume := cloudpath.Volume(example)
		path, sep := cloudpath.Path(example)
		parameters := cloudpath.Parameters(example)
		fmt.Printf("%v %q %q %q %q %c %v\n", local, scheme, host, volume, path, sep, parameters)
	}
	// Output:
	// false "s3" "" "my-bucket" "/my-bucket/object" / map[]
	// false "GoogleCloudStorage" "storage.cloud.google.com" "bucket" "/bucket/obj" / map[]
	// false "GoogleCloudStorage" "" "my-bucket" "/my-bucket" / map[]
	// true "windows" "localhost" "c" "c:\\root\\file" \ map[]
}

func ExamplePrefix() {
	date := cloudpath.Split("2012-11-27", '/').AsPrefix()
	for _, fullname := range []string{
		"s3://my-bucket/2012-11-27/shard-0000-of-0001.json",
		"/my-local-copy/2012-11-27/shard-0000-of-0001.json",
		"https://storage.cloud.google.com/google-copy/2012-11-27/shard-0001-of-0001.json",
	} {
		components := cloudpath.SplitPath(fullname)
		fmt.Printf("%v\n", components.Prefix().HasSuffix(date))
		// Output:
		// true
		// true
		// true
	}
}
