package evaluator

import (
	"monkey/ast"
	"monkey/object"
)

var (
	NULL  = &object.Null{}
	TRUE  = &object.Boolean{Value: true}
	FALSE = &object.Boolean{Value: false}
)

func Eval(node ast.Node) object.Object {
	switch node := node.(type) {
	// プログラム
	case *ast.Program:
		return evalProgram(node)

	// 式
	case *ast.ExpressionStatement:
		return Eval(node.Expression)

	// 整数リテラル
	case *ast.IntegerLiteral:
		return &object.Integer{Value: node.Value}

	// 真偽値
	case *ast.Boolean:
		return nativeBoolToBooleanObject(node.Value)

	// 前置式
	case *ast.PrefixExpression:
		right := Eval(node.Right)
		return evalPrefixExpression(node.Operator, right)

	// 中置式
	case *ast.InfixExpression:
		left := Eval(node.Left)
		right := Eval(node.Right)
		return evalInfixExpression(node.Operator, left, right)

	// ブロック文
	case *ast.BlockStatement:
		return evalBlockStatement(node)

	// if式
	case *ast.IfExpression:
		return evalIfExpression(node)

	// return文
	case *ast.ReturnStatement:
		val := Eval(node.ReturnValue)
		return &object.ReturnValue{Value: val}
	}

	return nil
}

/*
プログラムを評価
*/
func evalProgram(program *ast.Program) object.Object {
	var result object.Object

	for _, statement := range program.Statements {
		result = Eval(statement)

		if returnValue, ok := result.(*object.ReturnValue); ok {
			return returnValue.Value
		}
	}

	return result
}

/*
文を評価
*/
func evalStatements(stmts []ast.Statement) object.Object {
	var result object.Object

	for _, statement := range stmts {
		// 文を評価
		result = Eval(statement)

		// 文の評価結果が戻り値かどうか確認
		if returnValue, ok := result.(*object.ReturnValue); ok {
			// 戻り値の場合は以降の文の評価を打ち切り戻り値の値を返す
			return returnValue.Value
		}
	}

	return result
}

func evalPrefixExpression(operator string, right object.Object) object.Object {
	switch operator {
	case "!":
		return evalBangOperatorExpression(right)
	case "-":
		return evalMinusPrefixOperatorExpression(right)
	default:
		return NULL
	}
}

func evalBangOperatorExpression(right object.Object) object.Object {
	switch right {
	case TRUE:
		return FALSE
	case FALSE:
		return TRUE
	case NULL:
		return TRUE
	default:
		return FALSE
	}
}

func evalMinusPrefixOperatorExpression(right object.Object) object.Object {
	if right.Type() != object.INTEGER_OBJ {
		return NULL
	}

	value := right.(*object.Integer).Value
	return &object.Integer{Value: -value}

}

/*
中置式を評価
*/
func evalInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	switch {
	// 左辺、右辺共に整数の場合
	case left.Type() == object.INTEGER_OBJ && right.Type() == object.INTEGER_OBJ:
		// 整数同士を評価して結果を返す
		return evalIntegerInfixExpression(operator, left, right)
	//
	case operator == "==":
		return nativeBoolToBooleanObject(left == right)
	case operator == "!=":
		return nativeBoolToBooleanObject(left != right)
	default:
		return NULL
	}
}

/*
整数同士の中置式を評価
*/
func evalIntegerInfixExpression(
	operator string,
	left, right object.Object,
) object.Object {
	// 左辺の値を取り出す
	leftVal := left.(*object.Integer).Value
	// 右辺の値を取り出す
	rightVal := right.(*object.Integer).Value

	// 演算子で分岐
	switch operator {
	case "+":
		return &object.Integer{Value: leftVal + rightVal}
	case "-":
		return &object.Integer{Value: leftVal - rightVal}
	case "*":
		return &object.Integer{Value: leftVal * rightVal}
	case "/":
		return &object.Integer{Value: leftVal / rightVal}
	case "<":
		return nativeBoolToBooleanObject(leftVal < rightVal)
	case ">":
		return nativeBoolToBooleanObject(leftVal > rightVal)
	case "==":
		return nativeBoolToBooleanObject(leftVal == rightVal)
	case "!=":
		return nativeBoolToBooleanObject(leftVal != rightVal)
	default:
		return NULL
	}
}

func nativeBoolToBooleanObject(input bool) *object.Boolean {
	if input {
		return TRUE
	}
	return FALSE
}

func evalIfExpression(ie *ast.IfExpression) object.Object {
	condition := Eval(ie.Condition)

	if isTruthy(condition) {
		return Eval(ie.Consequence)
	} else if ie.Alternative != nil {
		return Eval(ie.Alternative)
	} else {
		return NULL
	}
}

/*
ブロック文を評価
*/
func evalBlockStatement(block *ast.BlockStatement) object.Object {
	var result object.Object

	// ノードに含まれている全ての文を評価
	for _, statement := range block.Statements {
		// 文を評価して結果を取得
		result = Eval(statement)

		// 文の結果がNULLでない＆戻り値オブジェクトの場合、以降の文の評価を打ち切り戻り値オブジェクトを返す
		// 戻り値の値ではなく、オブジェクトを返すことで、外側のブロック文でも評価を打ち切ってくれるようになる
		if result != nil && result.Type() == object.RETURN_VALUE_OBJ {
			return result
		}
	}

	return result
}

func isTruthy(obj object.Object) bool {
	switch obj {
	case NULL:
		return false
	case TRUE:
		return true
	case FALSE:
		return false
	default:
		return true
	}
}
