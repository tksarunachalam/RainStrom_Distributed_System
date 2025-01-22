package stream

// func Exec1(task m.Task) {

// 	filename := task.ParentJob.HyDFSSrcFile

// 	fileLineMap, err := ReadFilesToLines(filename, task)
// 	if err != nil {
// 		log.Printf("Error reading file: %v", err)
// 	}

// 	// Read the file from start to end filennumber in the given file and split the data in tuples of <word,1>
// 	for _, value := range fileLineMap {
// 		// Split the line into words
// 		words := strings.Fields(value)

// 		for _, word := range words {
// 			// TODO: Emit a tuple for each word and also write to a hydfs file
// 			// For example, you can print the tuple (in a real scenario, this would be passed to a further processing stage)
// 			fmt.Printf("(%s, 1)\n", word)
// 		}
// 	}
// }
