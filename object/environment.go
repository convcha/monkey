package object

func NewEnclosedEnvironment(outer *Environment) *Environment {
	env := NewEnvironment()
	env.outer = outer
	return env
}

/*
新規環境を生成
*/
func NewEnvironment() *Environment {
	s := make(map[string]Object)
	return &Environment{store: s}
}

/*
環境型
*/
type Environment struct {
	store map[string]Object
	outer *Environment
}

/*
指定された名前のオブジェクトを環境から取得
*/
func (e *Environment) Get(name string) (Object, bool) {
	obj, ok := e.store[name]
	if !ok && e.outer != nil {
		obj, ok = e.outer.Get(name)
	}
	return obj, ok
}

/*
環境にオブジェクトをセット
*/
func (e *Environment) Set(name string, val Object) Object {
	e.store[name] = val
	return val
}
