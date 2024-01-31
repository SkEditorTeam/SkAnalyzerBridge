package main

//#cgo CFLAGS: -g -Wall
import "C"

import (
	"log"
	"runtime"
	"tekao.net/jnigi"
)

func main() {}

var jvm *jnigi.JVM
var env *jnigi.Env
var skAnalyzer *jnigi.ObjectRef

//export InitJava
func InitJava(javaHome *C.char, analyzerJar *C.char) {
	if err := jnigi.LoadJVMLib(C.GoString(javaHome) + "/bin/server/jvm.dll"); err != nil {
		log.Fatal(err)
	}

	args := []string{"-Djava.class.path=" + C.GoString(analyzerJar)}
	var err error
	jvm, env, err = jnigi.CreateJVM(jnigi.NewJVMInitArgs(false, false, jnigi.JNI_VERSION_10, args))
	if err != nil {
		log.Fatal(err)
	}

	runtime.LockOSThread()
	jvm.AttachCurrentThread()
}

func getEnumField(class string, index int) *jnigi.ObjectRef {
	values := jnigi.NewObjectArrayRef(class)
	if err := env.CallStaticMethod(class, "values", values); err != nil {
		log.Fatal(err)
	}
	return env.FromObjectArray(values)[index]
}

type AnalyzerFlag uint8

const (
	ForceVaultHook AnalyzerFlag = 1 << iota
	ForceRegionsHook
)

func findAnalyzerFlags(analyzerFlags AnalyzerFlag) []*jnigi.ObjectRef {
	var flags []*jnigi.ObjectRef

	if analyzerFlags&ForceVaultHook != 0 {
		flags = append(flags, findAnalyzerFlag(0))
	}

	if analyzerFlags&ForceRegionsHook != 0 {
		flags = append(flags, findAnalyzerFlag(1))
	}

	return flags
}

func findAnalyzerFlag(analyzerFlag AnalyzerFlag) *jnigi.ObjectRef {
	return getEnumField("me/glicz/skanalyzer/AnalyzerFlag", int(analyzerFlag))
}

type LoggerType uint8

const (
	Disabled LoggerType = iota
	Normal
	Plain
)

func findLoggerType(loggerType LoggerType) *jnigi.ObjectRef {
	return getEnumField("me/glicz/skanalyzer/LoggerType", int(loggerType))
}

func javaFile(path *C.char) *jnigi.ObjectRef {
	if path == nil {
		return jnigi.NewObjectRef("java/io/File")
	}

	file, err := env.NewObject("java/io/File", c2javaString(path))
	if err != nil {
		log.Fatal(err)
	}
	return file
}

//export InitAnalyzer
func InitAnalyzer(analyzerFlags AnalyzerFlag, loggerType LoggerType, workingDir *C.char) {
	flags := env.ToObjectArray(findAnalyzerFlags(analyzerFlags), "me/glicz/skanalyzer/AnalyzerFlag")

	var err error
	skAnalyzer, err = env.NewObject("me/glicz/skanalyzer/SkAnalyzer", flags, findLoggerType(loggerType), javaFile(workingDir))
	if err != nil {
		log.Fatal(err)
	}
}

func c2javaString(str *C.char) *jnigi.ObjectRef {
	javaStr, err := env.NewObject("java/lang/String", []byte(C.GoString(str)))
	if err != nil {
		log.Fatal(err)
	}
	return javaStr
}

func java2cString(str *jnigi.ObjectRef) *C.char {
	var bytes []byte
	if err := str.CallMethod(env, "getBytes", &bytes); err != nil {
		log.Fatal(err)
	}
	return C.CString(string(bytes))
}

//export Parse
func Parse(path *C.char) *C.char {
	future := jnigi.NewObjectRef("java/util/concurrent/CompletableFuture")
	if err := skAnalyzer.CallMethod(env, "parseScript", future, c2javaString(path)); err != nil {
		log.Fatal(err)
	}

	joinResult := jnigi.NewObjectRef("java/lang/Object")
	if err := future.CallMethod(env, "join", joinResult); err != nil {
		log.Fatal(err)
	}

	scriptStructure := joinResult.Cast("me/glicz/skanalyzer/ScriptAnalyzeResult")

	jsonResult := jnigi.NewObjectRef("java/lang/String")
	if err := scriptStructure.GetField(env, "jsonResult", jsonResult); err != nil {
		log.Fatal(err)
	}

	return java2cString(jsonResult)
}

//export Exit
func Exit() {
	env.DeleteLocalRef(skAnalyzer)

	if err := jvm.DetachCurrentThread(env); err != nil {
		log.Fatal(err)
	}
	runtime.UnlockOSThread()

	if err := jvm.Destroy(); err != nil {
		log.Fatal(err)
	}
}
