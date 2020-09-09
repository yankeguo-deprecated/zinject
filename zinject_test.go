package zinject_test

import (
	"fmt"
	"github.com/zionkit/zinject"
	"reflect"
	"testing"
)

type SpecialString interface {
}

type TestStruct struct {
	Dep1 string        `inject:"" json:"-"`
	Dep2 SpecialString `inject:""`
	Dep3 string
}

type Greeter struct {
	Name string
}

func (g *Greeter) String() string {
	return "Hello, My name is" + g.Name
}

/* Test Helpers */
func expect(t *testing.T, a interface{}, b interface{}) {
	if a != b {
		t.Errorf("Expected %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func refute(t *testing.T, a interface{}, b interface{}) {
	if a == b {
		t.Errorf("Did not expect %v (type %v) - Got %v (type %v)", b, reflect.TypeOf(b), a, reflect.TypeOf(a))
	}
}

func Test_InjectorApply(t *testing.T) {
	injector := zinject.New()

	injector.Register("a dep", "").RegisterAs("another dep", "", (*SpecialString)(nil))

	s := TestStruct{}
	err := injector.Inject(&s)
	expect(t, err, nil)

	expect(t, s.Dep1, "a dep")
	expect(t, s.Dep2, "another dep")
	expect(t, s.Dep3, "")
}

func Test_InterfaceOf(t *testing.T) {
	iType := zinject.InterfaceOf((*SpecialString)(nil))
	expect(t, iType.Kind(), reflect.Interface)

	iType = zinject.InterfaceOf((**SpecialString)(nil))
	expect(t, iType.Kind(), reflect.Interface)

	// Expecting nil
	defer func() {
		rec := recover()
		refute(t, rec, nil)
	}()
	iType = zinject.InterfaceOf((*testing.T)(nil))
}

func Test_InjectorSet(t *testing.T) {
	injector := zinject.New()
	typ := reflect.TypeOf("string")
	typSend := reflect.ChanOf(reflect.SendDir, typ)
	typRecv := reflect.ChanOf(reflect.RecvDir, typ)

	// instantiating unidirectional channels is not possible using reflect
	// http://golang.org/src/pkg/reflect/value.go?s=60463:60504#L2064
	chanRecv := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, typ), 0)
	chanSend := reflect.MakeChan(reflect.ChanOf(reflect.BothDir, typ), 0)

	injector.Set(typSend, "", chanSend)
	injector.Set(typRecv, "", chanRecv)

	expect(t, injector.Get(typSend, "").IsValid(), true)
	expect(t, injector.Get(typRecv, "").IsValid(), true)
	expect(t, injector.Get(chanSend.Type(), "").IsValid(), false)
}

func Test_InjectorGet(t *testing.T) {
	injector := zinject.New()

	injector.Register("some dependency", "")

	expect(t, injector.Get(reflect.TypeOf("string"), "").IsValid(), true)
	expect(t, injector.Get(reflect.TypeOf(11), "").IsValid(), false)
}

func Test_InjectorSetParent(t *testing.T) {
	injector := zinject.New()
	injector.RegisterAs("another dep", "", (*SpecialString)(nil))

	injector2 := zinject.New()
	injector2.SetParent(injector)

	expect(t, injector2.Get(zinject.InterfaceOf((*SpecialString)(nil)), "").IsValid(), true)
}

func TestInjectImplementors(t *testing.T) {
	injector := zinject.New()
	g := &Greeter{"Jeremy"}
	injector.Register(g, "")

	expect(t, injector.Get(zinject.InterfaceOf((*fmt.Stringer)(nil)), "").IsValid(), true)
}
