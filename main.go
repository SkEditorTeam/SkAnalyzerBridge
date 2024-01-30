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

//export Init
func Init(javaHome *C.char, analyzerJar *C.char) {
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

	initAnalyzer()
}

func findAnalyzerFlag(flagName string) *jnigi.ObjectRef {
	flag := jnigi.NewObjectRef("me/glicz/skanalyzer/AnalyzerFlag")
	if err := env.GetStaticField("me/glicz/skanalyzer/AnalyzerFlag", flagName, flag); err != nil {
		log.Fatal(err)
	}
	return flag
}

func initAnalyzer() {
	flags := []*jnigi.ObjectRef{
		findAnalyzerFlag("FORCE_VAULT_HOOK"),
		findAnalyzerFlag("FORCE_REGIONS_HOOK"),
		findAnalyzerFlag("ENABLE_PLAIN_LOGGER"),
	}
	flagsObjectArray := env.ToObjectArray(flags, "me/glicz/skanalyzer/AnalyzerFlag")

	var err error
	skAnalyzer, err = env.NewObject("me/glicz/skanalyzer/SkAnalyzer", flagsObjectArray, jnigi.NewObjectRef("java/io/File"))
	if err != nil {
		log.Fatal(err)
	}
}

func javaString(str string) *jnigi.ObjectRef {
	javaStr, err := env.NewObject("java/lang/String", []byte(str))
	if err != nil {
		log.Fatal(err)
	}
	return javaStr
}

//export Parse
func Parse(path *C.char) {
	future := jnigi.NewObjectRef("java/util/concurrent/CompletableFuture")
	if err := skAnalyzer.CallMethod(env, "parseScript", future, javaString(C.GoString(path))); err != nil {
		log.Fatal(err)
	}
	env.DeleteLocalRef(future)
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
