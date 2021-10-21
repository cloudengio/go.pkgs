7a8,9
> 	"os"
> 	// "os"
15,17d16
< // deletion 1
< // deletion 2
< 
31a31,42
> /*
> func WordDecoder(data []byte) (string, int) {
> 	idx := bytes.IndexFunc(data, unicode.IsSpace)
> 	if idx < 0 {
> 		return "", 0
> 	}
> 	h := fnv.New64a()
> 	h.Write(data[:idx])
> 	sum := h.Sum64()
> 	return string(data[:idx]), int64(sum), idx + 1
> }*/
> 
37a49
> 	fmt.Printf("D: %v %v\n", len(ld.lines), line)
118a131
> 		fmt.Printf("A: %v\n", linesA)
132c145
< 		//lcs.PrettyHorizontal(os.Stdout, []int32(g.textA), g.edits)
---
> 		lcs.PrettyHorizontal(os.Stdout, []int32(g.textA), g.edits)
137,139c150,151
< // multiline change
< // multiline change
< // multiline change
---
> // multiline change 1
> // multiline change 2
