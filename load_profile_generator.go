package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func readFileContent(filePath string) (string, error) {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Fatalf("failed reading file [%s]: %s", filePath, err)
	}
	return string(content), err
}

func overwriteFileContent(filePath string, newContent string) error {
	file, err := os.OpenFile(filePath, os.O_RDWR, 0777)

	if err != nil {
		log.Fatalf("failed opening file: %s", err)
	}
	defer file.Close()

	len, err := file.WriteAt([]byte(newContent), 0) // Write at 0 beginning
	if err != nil {
		log.Fatalf("failed writing to file: %s", err)
	}
	fmt.Printf("\nLength: %d bytes", len)
	fmt.Printf("\nFile Name: %s\n", file.Name())
	return err
}

func createLoadProfile(initLoad, increment, rampUp, rampUpPropName, stepDuration, stepDurationPropName, numSteps int, loadStepPattern string, stdout strings.Builder) string {
	for i := 0; i < numSteps; i++ {
		var prev int
		var next int
		if i == 0 {
			prev = initLoad
			next = initLoad + increment
		} else {
			prev = initLoad + (increment * i)
			next = prev + increment
		}
		stdout.WriteString(fmt.Sprintf(loadStepPattern, time.Now().UnixNano(), prev, prev, next, next, rampUpPropName, rampUp))
		stdout.WriteString("\n")
		stdout.WriteString(fmt.Sprintf(loadStepPattern, time.Now().UnixNano(), next, next, next, next, stepDurationPropName, stepDuration))
		stdout.WriteString("\n")
	}
	stdout.WriteString(`          </collectionProp>`)
	stdout.WriteString("\n")
	stdout.WriteString(`      	</kg`)
	return stdout.String()
}

func getInputArgs() []string {
	argsString := ""
	if len(os.Args) > 1 {
		argsString = strings.Join(os.Args[1:], " ")
	} else {
		fmt.Println("set next args: jmxPath, initLoad,increment,rampUp,stepDuration,numSteps")
		fmt.Println("initLoad,increment,rampUp,stepDuration,numSteps must be gt 0")
		os.Exit(0)
	}
	args := strings.Split(argsString, " ")
	return args
}

func getLoadStepPattern() string {
	var builder strings.Builder
	builder.WriteString(`            <collectionProp name="%d">`)
	builder.WriteString("\n")
	builder.WriteString(`		<stringProp name="%d">%d</stringProp>`)
	builder.WriteString("\n")
	builder.WriteString(`		<stringProp name="%d">%d</stringProp>`)
	builder.WriteString("\n")
	builder.WriteString(`		<stringProp name="%d">%d</stringProp>`)
	builder.WriteString("\n")
	builder.WriteString(`	    </collectionProp>`)
	return builder.String()
}

func main() {

	inputArgs := getInputArgs()

	jmxPath := "NOT_FOUND"
	initLoad := 0
	increment := 0
	rampUp := 0
	stepDuration := 0
	numSteps := 0
	rampUpPropName := 1000000
	stepDurationPropName := 1000000
	totalTestDuration := 0

	switch len(inputArgs) {
	case 1, 2, 3, 4, 5:
		fmt.Println("set next args: jmxPath, initLoad,increment,rampUp,stepDuration,numSteps")
		fmt.Println("initLoad,increment,rampUp,stepDuration,numSteps must be gt 0")
		os.Exit(0)
	case 6:
		jmxPath = inputArgs[0]
		initLoad, _ = strconv.Atoi(inputArgs[1])
		increment, _ = strconv.Atoi(inputArgs[2])
		rampUp, _ = strconv.Atoi(inputArgs[3])
		rampUpPropName += rampUp
		stepDuration, _ = strconv.Atoi(inputArgs[4])
		stepDurationPropName += stepDuration
		numSteps, _ = strconv.Atoi(inputArgs[5])
		totalTestDuration = numSteps*(rampUp+stepDuration) + 5
	}

	content, _ := readFileContent(jmxPath)

	loadStepPattern := getLoadStepPattern()

	var stdout strings.Builder

	newLoadProfile := createLoadProfile(initLoad, increment, rampUp, rampUpPropName, stepDuration, stepDurationPropName, numSteps, loadStepPattern, stdout)

	loadProfMatcher := regexp.MustCompile(`\<collectionProp name="[^load_profile]*"\>(\s*|.*)*\<\/kg`) // `<collectionProp name="[^load_profile]">(.|\s)*?<\/kg`
	newContent := loadProfMatcher.ReplaceAllLiteralString(content, newLoadProfile)

	loadProfDurationPattern := `<stringProp name="Hold">%d</stringProp>`
	loadProfDurationMatcher := regexp.MustCompile(`<stringProp name="Hold">(\d*?)<\/stringProp>`)
	newContent = loadProfDurationMatcher.ReplaceAllLiteralString(newContent, fmt.Sprintf(loadProfDurationPattern, totalTestDuration))
	fmt.Println(newContent)
	overwriteFileContent(jmxPath, newContent)

}
