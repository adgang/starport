package astutils

// import (
// 	"fmt"
// 	"os"

// 	"github.com/oshothebig/pbast"
// 	"github.com/oshothebig/pbast/printer"
// )

// type PBAstHelper struct {
// 	astFile *pbast.File
// 	file    string
// }

// func NewPBAstHelper(file string) *PBAstHelper {
// 	pbastHelper := new(PBAstHelper)

// 	pbastFile := pbast.NewFile(pbast.Package(file))

// 	pbastHelper.astFile = pbastFile
// 	pbastHelper.file = file
// 	pbastHelper.Print()

// 	return pbastHelper
// }

// func (pbastHelper *PBAstHelper) Close() {
// 	// TODO: cleanup the resources
// }

// func (pbastHelper *PBAstHelper) AddImport(pkg string) (done bool) {
// 	astFile := pbastHelper.astFile
// 	for _, importPkg := range astFile.Imports {
// 		if importPkg.Name == pkg {
// 			fmt.Println("import of " + pkg + " already added")
// 			return false
// 		}
// 	}
// 	astFile.Imports = append(astFile.Imports, pbast.NewImport(pkg))
// 	return true
// }

// func (pbastHelper *PBAstHelper) Print() {

// 	printer.Fprint(os.Stdout, pbastHelper.astFile)
// }

// func (pbastHelper *PBAstHelper) Write() error {

// 	f, err := os.Open(pbastHelper.file)

// 	if err != nil {
// 		return err
// 	}
// 	printer.Fprint(f, pbastHelper.astFile)
// 	return err

// }
