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

func createLoadProfile(l loadProfile, loadStepPattern string) string {
	var stdout strings.Builder
	for i := 0; i < l.numSteps; i++ {
		var prev int
		var next int
		if i == 0 {
			prev = l.initLoad
			next = l.initLoad + l.increment
		} else {
			prev = l.initLoad + (l.increment * i)
			next = prev + l.increment
		}
		stdout.WriteString(fmt.Sprintf(loadStepPattern, time.Now().UnixNano(), prev, prev, next, next, l.rampUpPropName, l.rampUp))
		stdout.WriteString("\n")
		stdout.WriteString(fmt.Sprintf(loadStepPattern, time.Now().UnixNano(), next, next, next, next, l.stepDurationPropName, l.stepDuration))
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

func assignLoadProfileParams(inputArgs []string, l *loadProfile) {
	switch len(inputArgs) {
	case 1, 2, 3, 4, 5:
		fmt.Println("set next args: jmxPath, initLoad,increment,rampUp,stepDuration,numSteps")
		fmt.Println("initLoad,increment,rampUp,stepDuration,numSteps must be gt 0")
		os.Exit(0)
	case 6:
		l.jmxPath = inputArgs[0]
		fmt.Println(l.jmxPath)
		l.initLoad, _ = strconv.Atoi(inputArgs[1])
		l.increment, _ = strconv.Atoi(inputArgs[2])
		l.rampUp, _ = strconv.Atoi(inputArgs[3])
		l.rampUpPropName = l.rampUpPropName + l.rampUp
		l.stepDuration, _ = strconv.Atoi(inputArgs[4])
		l.stepDurationPropName = l.stepDurationPropName + l.stepDuration
		l.numSteps, _ = strconv.Atoi(inputArgs[5])
		l.totalTestDuration = l.numSteps*(l.rampUp+l.stepDuration) + 5
	}
}

type loadProfile struct {
	jmxPath              string
	initLoad             int
	increment            int
	rampUp               int
	stepDuration         int
	numSteps             int
	rampUpPropName       int
	stepDurationPropName int
	totalTestDuration    int
}

func main() {

	inputArgs := getInputArgs()

	loadP := loadProfile{
		jmxPath:              "NOT_FOUND",
		initLoad:             0,
		increment:            0,
		rampUp:               0,
		stepDuration:         0,
		numSteps:             0,
		rampUpPropName:       1000000,
		stepDurationPropName: 1000000,
		totalTestDuration:    0,
	}

	assignLoadProfileParams(inputArgs, &loadP)

	fmt.Println(loadP.jmxPath)

	oldContent, _ := readFileContent(loadP.jmxPath)

	loadStepPattern := getLoadStepPattern()

	newLoadProfile := createLoadProfile(loadP, loadStepPattern)
	loadProfMatcher := regexp.MustCompile(`\<collectionProp name="[^load_profile]*"\>(\s*|.*)*\<\/kg`) // `<collectionProp name="[^load_profile]">(.|\s)*?<\/kg`

	newContent := loadProfMatcher.ReplaceAllLiteralString(oldContent, newLoadProfile)

	loadProfDurationPattern := `<stringProp name="Hold">%d</stringProp>`
	loadProfDurationMatcher := regexp.MustCompile(`<stringProp name="Hold">(\d*?)<\/stringProp>`)

	newContent = loadProfDurationMatcher.ReplaceAllLiteralString(newContent, fmt.Sprintf(loadProfDurationPattern, loadP.totalTestDuration))
	//fmt.Println(newContent)
	overwriteFileContent(loadP.jmxPath, newContent)

}

