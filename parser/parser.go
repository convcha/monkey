package parser

import (
	"fmt"
	"monkey/ast"
	"monkey/lexer"
	"monkey/token"
	"strconv"
)

type Parser struct {
	l *lexer.Lexer

	curToken  token.Token
	peekToken token.Token
	errors    []string

	prefixParseFns map[token.TokenType]prefixParseFn
	infixParseFns  map[token.TokenType]infixParseFn
}

type (
	prefixParseFn func() ast.Expression
	infixParseFn  func(ast.Expression) ast.Expression
)

const (
	_ int = iota
	LOWEST
	EQUALS      // ==
	LESSGREATER // > または <
	SUM         // +
	PRODUCT     // *
	PREFIX      // -X または !X
	CALL        // myFunction(X)
)

var precedences = map[token.TokenType]int{
	token.EQ:       EQUALS,
	token.NOT_EQ:   EQUALS,
	token.LT:       LESSGREATER,
	token.GT:       LESSGREATER,
	token.PLUS:     SUM,
	token.MINUS:    SUM,
	token.SLASH:    PRODUCT,
	token.ASTERISK: PRODUCT,
	token.LPAREN:   CALL,
}

func New(l *lexer.Lexer) *Parser {
	p := &Parser{l: l, errors: []string{}}

	p.prefixParseFns = make(map[token.TokenType]prefixParseFn)
	p.registerPrefix(token.IDENT, p.parseIdentifier)
	p.registerPrefix(token.INT, p.parseIntegerLiteral)
	p.registerPrefix(token.BANG, p.parsePrefixExpression)
	p.registerPrefix(token.MINUS, p.parsePrefixExpression)
	p.registerPrefix(token.TRUE, p.parseBoolean)
	p.registerPrefix(token.FALSE, p.parseBoolean)
	p.registerPrefix(token.LPAREN, p.parseGroupedExpression)
	p.registerPrefix(token.IF, p.parseIfExpression)
	p.registerPrefix(token.FUNCTION, p.parseFunctionLiteral)

	p.infixParseFns = make(map[token.TokenType]infixParseFn)
	p.registerInfix(token.PLUS, p.parseInfixExpression)
	p.registerInfix(token.MINUS, p.parseInfixExpression)
	p.registerInfix(token.SLASH, p.parseInfixExpression)
	p.registerInfix(token.ASTERISK, p.parseInfixExpression)
	p.registerInfix(token.EQ, p.parseInfixExpression)
	p.registerInfix(token.NOT_EQ, p.parseInfixExpression)
	p.registerInfix(token.LT, p.parseInfixExpression)
	p.registerInfix(token.GT, p.parseInfixExpression)
	p.registerInfix(token.LPAREN, p.parseCallExpression)

	// 2つトークンを読み込む。curTokenとpeekTokenの両方がセットされる。
	p.nextToken()
	p.nextToken()

	return p
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.peekToken = p.l.NextToken()
}

// プログラムを解析
func (p *Parser) ParseProgram() *ast.Program {
	program := &ast.Program{}
	program.Statements = []ast.Statement{}

	for p.curToken.Type != token.EOF {
		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
		p.nextToken()
	}

	return program
}

// 文を解析
func (p *Parser) parseStatement() ast.Statement {
	switch p.curToken.Type {
	case token.LET:
		return p.parseLetStatement()
	case token.RETURN:
		return p.parseReturnStatement()
	default:
		return p.parseExpressionStatement()
	}
}

/*
let文を解析
*/
func (p *Parser) parseLetStatement() *ast.LetStatement {
	stmt := &ast.LetStatement{Token: p.curToken}

	if !p.expectPeek(token.IDENT) {
		return nil
	}

	stmt.Name = &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}

	if !p.expectPeek(token.ASSIGN) {
		return nil
	}

	p.nextToken()

	stmt.Value = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

/*
return文を解析
*/
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.curToken}

	p.nextToken()

	stmt.ReturnValue = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// 式文を解析
func (p *Parser) parseExpressionStatement() *ast.ExpressionStatement {
	defer untrace(trace("parseExpressionStatement"))

	stmt := &ast.ExpressionStatement{Token: p.curToken}

	stmt.Expression = p.parseExpression(LOWEST)

	if p.peekTokenIs(token.SEMICOLON) {
		p.nextToken()
	}

	return stmt
}

// 前置構文解析関数エラー
func (p *Parser) noPrefixParseFnError(t token.TokenType) {
	msg := fmt.Sprintf("no prefix parse function for %s found", t)
	p.errors = append(p.errors, msg)
}

// 式を解析
func (p *Parser) parseExpression(precedence int) ast.Expression {
	defer untrace(trace("parseExpression"))

	prefix := p.prefixParseFns[p.curToken.Type]
	if prefix == nil {
		p.noPrefixParseFnError(p.curToken.Type)
		return nil
	}
	leftExp := prefix()

	for !p.peekTokenIs(token.SEMICOLON) && precedence < p.peekPrecedence() {
		infix := p.infixParseFns[p.peekToken.Type]
		if infix == nil {
			return leftExp
		}

		p.nextToken()

		leftExp = infix(leftExp)
	}

	return leftExp
}

// 整数リテラルを解析
func (p *Parser) parseIntegerLiteral() ast.Expression {
	defer untrace(trace("parseIntegerLiteral"))

	lit := &ast.IntegerLiteral{Token: p.curToken}

	value, err := strconv.ParseInt(p.curToken.Literal, 0, 64)
	if err != nil {
		msg := fmt.Sprintf("could not parse %q as integer", p.curToken.Literal)
		p.errors = append(p.errors, msg)
		return nil
	}

	lit.Value = value

	return lit
}

func (p *Parser) parsePrefixExpression() ast.Expression {
	defer untrace(trace("parsePrefixExpression"))

	expression := &ast.PrefixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
	}

	p.nextToken()
	expression.Right = p.parseExpression(PREFIX)

	return expression
}

// 中置式を解析
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	defer untrace(trace("parseInfixExpression"))

	expression := &ast.InfixExpression{
		Token:    p.curToken,
		Operator: p.curToken.Literal,
		Left:     left,
	}

	precedence := p.curPrecedence()
	p.nextToken()
	expression.Right = p.parseExpression(precedence)

	return expression
}

// 真偽値リテラルを解析
func (p *Parser) parseBoolean() ast.Expression {
	return &ast.Boolean{Token: p.curToken, Value: p.curTokenIs(token.TRUE)}
}

// グループ化された式を解析
func (p *Parser) parseGroupedExpression() ast.Expression {
	p.nextToken()

	exp := p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	return exp
}

// if式を解析
func (p *Parser) parseIfExpression() ast.Expression {
	expression := &ast.IfExpression{Token: p.curToken}

	if !p.expectPeek(token.LPAREN) {
		return nil
	}

	p.nextToken()
	expression.Condition = p.parseExpression(LOWEST)

	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	if !p.expectPeek(token.LBRACE) {
		return nil
	}

	expression.Consequence = p.parseBlockStatement()

	if p.peekTokenIs(token.ELSE) {
		p.nextToken()

		if !p.expectPeek(token.LBRACE) {
			return nil
		}

		expression.Alternative = p.parseBlockStatement()
	}

	return expression
}

/*
呼び出し式を解析
*/
func (p *Parser) parseCallExpression(function ast.Expression) ast.Expression {
	// 呼び出し式ノードを生成
	exp := &ast.CallExpression{Token: p.curToken, Function: function}
	// 引数を解析＆解析結果を呼び出し式ノードの引数リストにセット
	exp.Arguments = p.parseCallArguments()
	// 生成した呼び出し式ノードを返す
	return exp
}

/*
呼び出し式の引数を解析
*/
func (p *Parser) parseCallArguments() []ast.Expression {
	// 引数リスト(式ノードの配列)を定義
	args := []ast.Expression{}

	// 次のトークンが右丸カッコかどうかチェック。右丸カッコでない場合、パラメータ無しとわかる。
	if p.peekTokenIs(token.RPAREN) {
		// トークンを一つ進める。右丸カッコがカレントになる。
		p.nextToken()
		// 空の引数リストを返す
		return args
	}

	// トークンを一つ進める。一つ目の引数(式)がカレントになる。
	p.nextToken()
	// 一つ目の引数(式)を解析＆解析結果(式ノード)を引数リストに追加。
	args = append(args, p.parseExpression(LOWEST))

	// 次のトークンがカンマである間ループさせる
	for p.peekTokenIs(token.COMMA) {
		// トークンを一つ進める。カンマがカレントになる。
		p.nextToken()
		// トークンを一つ進める。次の引数(式)がカレントになる。
		p.nextToken()
		// 次の引数(式)を解析＆解析結果(式ノード)を引数リストに追加。
		args = append(args, p.parseExpression(LOWEST))
	}

	// 全ての引数を解析した後、次のトークンが右丸カッコでなかったら何も返さない(構文解析エラー)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	// 生成した引数リスト(式ノードの配列)を返す
	return args
}

/*
ブロック文を解析
*/
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{Token: p.curToken}
	block.Statements = []ast.Statement{}

	p.nextToken()

	for !p.curTokenIs(token.RBRACE) && !p.curTokenIs(token.EOF) {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.nextToken()
	}

	return block
}

/*
関数リテラルを解析
*/
func (p *Parser) parseFunctionLiteral() ast.Expression {
	// 関数リテラルノードを生成
	lit := &ast.FunctionLiteral{Token: p.curToken}

	// 次のトークンが左丸カッコでなかったら何も返さない(構文解析エラー)
	if !p.expectPeek(token.LPAREN) {
		return nil
	} // トークンを一つ進める。左丸カッコがカレントになる。

	// パラメータを解析＆関数リテラルノードのパラメータリストにセット
	lit.Parameters = p.parseFunctionParameters()

	// 次のトークンが左中カッコかどうかチェック。左中カッコでない場合、関数本体のブロック文が不正なので何も返さない(構文解析エラー)
	if !p.expectPeek(token.LBRACE) {
		return nil
	} // トークンを一つ進める。左中カッコがカレントになる。

	// 関数本体(ブロック文)を解析＆関数リテラルノードの本体にセット
	lit.Body = p.parseBlockStatement()

	// 生成した関数リテラルノードを返す。これにはパラメータリストと本体が含まれている。
	return lit
}

/*
関数パラメータを解析
*/
func (p *Parser) parseFunctionParameters() []*ast.Identifier {
	// パラメータリストを定義
	identifiers := []*ast.Identifier{}

	// 次のトークンが右丸カッコかどうかチェック。右丸カッコでない場合、パラメータ無しとわかる。
	if p.peekTokenIs(token.RPAREN) {
		// トークンを一つ進める。右丸カッコがカレントになる。
		p.nextToken()
		// 空のパラメータリストを返す
		return identifiers
	}

	// トークンを一つ進める。一つ目のパラメータがカレントになる。
	p.nextToken()

	// 一つ目のパラメータのノード(識別子)を生成
	ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
	// 一つ目のパラメータをパラメータリストに追加
	identifiers = append(identifiers, ident)

	// 次のトークンがカンマである間ループさせる
	for p.peekTokenIs(token.COMMA) {
		// トークンを一つ進める。カンマがカレントになる。
		p.nextToken()
		// トークンを一つ進める。次のパラメータがカレントになる。
		p.nextToken()
		// 次のパラメータのノード(識別子)を生成
		ident := &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
		// 次のパラメータをパラメータリストに追加
		identifiers = append(identifiers, ident)
	}

	// 全てのパラメータを解析した後、次のトークンが右丸カッコでなかったら何も返さない(構文解析エラー)
	if !p.expectPeek(token.RPAREN) {
		return nil
	}

	// パラメータリストを返す
	return identifiers
}

func (p *Parser) curTokenIs(t token.TokenType) bool {
	return p.curToken.Type == t
}

func (p *Parser) peekTokenIs(t token.TokenType) bool {
	return p.peekToken.Type == t
}

func (p *Parser) expectPeek(t token.TokenType) bool {
	if p.peekTokenIs(t) {
		p.nextToken()
		return true
	} else {
		p.peekError(t)
		return false
	}
}

func (p *Parser) Errors() []string {
	return p.errors
}

func (p *Parser) peekError(t token.TokenType) {
	msg := fmt.Sprintf("expected next token to be %s, got %s instead",
		t, p.peekToken.Type)
	p.errors = append(p.errors, msg)
}

func (p *Parser) registerPrefix(tokenType token.TokenType, fn prefixParseFn) {
	p.prefixParseFns[tokenType] = fn
}

func (p *Parser) registerInfix(tokenType token.TokenType, fn infixParseFn) {
	p.infixParseFns[tokenType] = fn
}

func (p *Parser) parseIdentifier() ast.Expression {
	return &ast.Identifier{Token: p.curToken, Value: p.curToken.Literal}
}

func (p *Parser) peekPrecedence() int {
	if p, ok := precedences[p.peekToken.Type]; ok {
		return p
	}

	return LOWEST
}

func (p *Parser) curPrecedence() int {
	if p, ok := precedences[p.curToken.Type]; ok {
		return p
	}

	return LOWEST
}
