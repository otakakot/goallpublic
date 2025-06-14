package main

import (
	"fmt"
	"go/ast"
	"go/doc"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <path or file>")
		os.Exit(1)
	}

	path := os.Args[1]
	files := []string{}

	// ファイルまたはディレクトリを探索
	filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// vendorディレクトリを除外
		if strings.Contains(path, "vendor") {
			return nil
		}

		// ファイルのみ対象
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			files = append(files, path)
		}
		return nil
	})

	for _, file := range files {
		processFile(file)
	}
}

func processFile(filePath string) {
	// ファイルを開く
	content, err := os.ReadFile(filePath)
	if err != nil {
		fmt.Printf("Failed to read file: %s\n", filePath)
		return
	}

	// "DO NOT EDIT." を含むファイルを除外
	if strings.Contains(string(content), "DO NOT EDIT.") {
		return
	}

	// 構文解析
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, content, parser.AllErrors)
	if err != nil {
		fmt.Printf("Failed to parse file: %s\n", filePath)
		return
	}

	// ASTを修正
	changed := false
	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			// main関数は除外
			if x.Name.Name == "main" {
				return true
			}
			// メソッド名を大文字に変換
			if x.Recv != nil && isLowerCase(x.Name.Name) {
				oldName := x.Name.Name
				x.Name.Name = capitalize(x.Name.Name)
				changed = true

				// メソッド呼び出し側も変更
				ast.Inspect(node, func(n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok {
						if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
							if sel.Sel.Name == oldName {
								sel.Sel.Name = capitalize(sel.Sel.Name)
								changed = true
							}
						}
					}
					return true
				})
			}
			// 関数名を大文字に変換
			if isLowerCase(x.Name.Name) {
				oldName := x.Name.Name
				x.Name.Name = capitalize(x.Name.Name)
				changed = true

				// 関数呼び出し側も変更
				ast.Inspect(node, func(n ast.Node) bool {
					if call, ok := n.(*ast.CallExpr); ok {
						if ident, ok := call.Fun.(*ast.Ident); ok && ident.Name == oldName {
							ident.Name = capitalize(ident.Name)
							changed = true
						}
					}
					return true
				})
			}
			// レシーバーの型名を大文字に変換
			if x.Recv != nil {
				for _, field := range x.Recv.List {
					if ident, ok := field.Type.(*ast.Ident); ok && isLowerCase(ident.Name) {
						ident.Name = capitalize(ident.Name)
						changed = true
					}
				}
			}
		case *ast.CallExpr:
			// 関数呼び出しを大文字に変換
			if fun, ok := x.Fun.(*ast.Ident); ok && isLowerCase(fun.Name) {
				fun.Name = capitalize(fun.Name)
				changed = true
			}
		case *ast.InterfaceType:
			// インタフェースのメソッド名を大文字に変換
			if x.Methods != nil {
				for _, method := range x.Methods.List {
					for _, name := range method.Names {
						if isLowerCase(name.Name) {
							name.Name = capitalize(name.Name)
							changed = true
						}
					}
				}
			}
		case *ast.TypeSpec:
			// 型名を大文字に変換
			if x.Name != nil && isLowerCase(x.Name.Name) {
				x.Name.Name = capitalize(x.Name.Name)
				changed = true
			}
			// インタフェース名を大文字に変換
			if x.Type != nil {
				if _, ok := x.Type.(*ast.InterfaceType); ok {
					if isLowerCase(x.Name.Name) {
						x.Name.Name = capitalize(x.Name.Name)
						changed = true
					}
				}
			}
		case *ast.StructType:
			// 構造体のフィールド名を大文字に変換
			if x.Fields != nil {
				for _, field := range x.Fields.List {
					for _, name := range field.Names {
						if isLowerCase(name.Name) {
							name.Name = capitalize(name.Name)
							changed = true
						}
					}
				}
			}
		case *ast.CompositeLit:
			// 構造体リテラルの型名とキーを大文字に変換
			if typ, ok := x.Type.(*ast.Ident); ok {
				if isLowerCase(typ.Name) {
					typ.Name = capitalize(typ.Name)
					changed = true
				}
			}
			for _, elt := range x.Elts {
				if kv, ok := elt.(*ast.KeyValueExpr); ok {
					if key, ok := kv.Key.(*ast.Ident); ok && isLowerCase(key.Name) {
						key.Name = capitalize(key.Name)
						changed = true
					}
				}
			}
		case *ast.SelectorExpr:
			// セレクタ式のフィールド名を大文字に変換
			if x.Sel != nil && isLowerCase(x.Sel.Name) {
				x.Sel.Name = capitalize(x.Sel.Name)
				changed = true
			}
		case *ast.ValueSpec:
			// グローバル変数と定数の名前を大文字に変換
			for _, name := range x.Names {
				if isLowerCase(name.Name) {
					name.Name = capitalize(name.Name)
					changed = true
				}
			}
			// 変数宣言で使用されている型名を大文字に変換
			if x.Type != nil {
				if ident, ok := x.Type.(*ast.Ident); ok && isLowerCase(ident.Name) {
					// 予約後の型は除外
					if !isReservedType(ident.Name) {
						ident.Name = capitalize(ident.Name)
						changed = true
					}
				}
			}
			for _, value := range x.Values {
				ast.Inspect(value, func(n ast.Node) bool {
					if ident, ok := n.(*ast.Ident); ok && isLowerCase(ident.Name) {
						// 予約後の型は除外
						if !isReservedType(ident.Name) {
							ident.Name = capitalize(ident.Name)
							changed = true
						}
					}
					return true
				})
			}
		}
		return true
	})

	if changed {
		// フォーマット
		var output strings.Builder
		printer.Fprint(&output, fset, node)
		formatted, err := format.Source([]byte(output.String()))
		if err != nil {
			fmt.Printf("Failed to format file: %s\n", filePath)
			return
		}

		// ファイルに書き戻す
		os.WriteFile(filePath, formatted, 0644)
		fmt.Printf("Modified: %s\n", filePath)
	}
}

func isLowerCase(s string) bool {
	return len(s) > 0 && s[0] >= 'a' && s[0] <= 'z'
}

func capitalize(s string) string {
	if len(s) == 0 {
		return s
	}
	return strings.ToUpper(string(s[0])) + s[1:]
}

func isReservedType(name string) bool {
	// go/tokenパッケージを利用して予約語を判定
	if token.Lookup(name).IsKeyword() {
		return true
	}
	// go/docパッケージを利用してプリでクレアド識別子の型を判定
	if doc.IsPredeclared(name) {
		return true
	}
	return false
}
