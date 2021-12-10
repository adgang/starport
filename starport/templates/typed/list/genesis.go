package list

import (
	"fmt"
	"path/filepath"

	"github.com/gobuffalo/genny"
	"github.com/tendermint/starport/starport/pkg/astutils"
	"github.com/tendermint/starport/starport/pkg/placeholder"

	"github.com/tendermint/starport/starport/templates/module"
	"github.com/tendermint/starport/starport/templates/typed"
)

func genesisModify(replacer placeholder.Replacer, opts *typed.Options, g *genny.Generator) {
	g.RunFn(genesisProtoModify(replacer, opts))
	g.RunFn(genesisTypesModify(opts))
	g.RunFn(genesisModuleModify(opts))
	g.RunFn(genesisTestsModify(opts))
	g.RunFn(genesisTypesTestsModify(replacer, opts))
}

func genesisProtoModify(replacer placeholder.Replacer, opts *typed.Options) genny.RunFn {
	return func(r *genny.Runner) error {
		path := filepath.Join(opts.AppPath, "proto", opts.ModuleName, "genesis.proto")
		f, err := r.Disk.Find(path)
		if err != nil {
			return err
		}

		templateProtoImport := `import "%[2]v/%[3]v.proto";
%[1]v`
		replacementProtoImport := fmt.Sprintf(
			templateProtoImport,
			typed.PlaceholderGenesisProtoImport,
			opts.ModuleName,
			opts.TypeName.Snake,
		)
		content := replacer.Replace(f.String(), typed.PlaceholderGenesisProtoImport, replacementProtoImport)

		// Add gogo.proto
		replacementGogoImport := typed.EnsureGogoProtoImported(path, typed.PlaceholderGenesisProtoImport)
		content = replacer.Replace(content, typed.PlaceholderGenesisProtoImport, replacementGogoImport)

		// Parse proto file to determine the field numbers
		highestNumber, err := typed.GenesisStateHighestFieldNumber(path)
		if err != nil {
			return err
		}

		templateProtoState := `repeated %[2]v %[3]vList = %[4]v [(gogoproto.nullable) = false];
  uint64 %[3]vCount = %[5]v;
  %[1]v`
		replacementProtoState := fmt.Sprintf(
			templateProtoState,
			typed.PlaceholderGenesisProtoState,
			opts.TypeName.UpperCamel,
			opts.TypeName.LowerCamel,
			highestNumber+1,
			highestNumber+2,
		)
		content = replacer.Replace(content, typed.PlaceholderGenesisProtoState, replacementProtoState)

		newFile := genny.NewFileS(path, content)
		return r.File(newFile)
	}
}

func genesisTypesModify(opts *typed.Options) genny.RunFn {

	return func(r *genny.Runner) error {
		path := filepath.Join(opts.AppPath, "x", opts.ModuleName, "types/genesis.go")

		dstHelper, err := astutils.NewDstHelper(path)
		defer dstHelper.Close()

		if err != nil {
			return err
		}

		_, err = dstHelper.AddImport("fmt")
		if err != nil {
			return err
		}

		key := fmt.Sprintf(`%[1]vList`, opts.TypeName.UpperCamel)
		typeName := fmt.Sprintf(`%[1]v`, opts.TypeName.UpperCamel)

		typed.AddKeysToDefaultGenesisState(dstHelper, key, typeName)

		templateTypesValidate := `// Check for duplicated ID in %[1]v
%[1]vIdMap := make(map[uint64]bool)
%[2]vCount := gs.Get%[2]vCount()
for _, elem := range gs.%[2]vList {
	if _, ok := %[1]vIdMap[elem.Id]; ok {
		return fmt.Errorf("duplicated id for %[1]v")
	}
	if elem.Id >= %[1]vCount {
		return fmt.Errorf("%[1]v id should be lower or equal than the last id")
	}
	%[1]vIdMap[elem.Id] = true
}
`
		replacementTypesValidate := fmt.Sprintf(
			templateTypesValidate,
			opts.TypeName.LowerCamel,
			opts.TypeName.UpperCamel,
		)

		typed.AddGenesisStateValidation(dstHelper, replacementTypesValidate)

		content, err := dstHelper.Content()

		if err != nil {
			return err
		}

		newFile := genny.NewFileS(path, content)
		return r.File(newFile)
	}
}

func genesisModuleModify(opts *typed.Options) genny.RunFn {
	return func(r *genny.Runner) error {
		path := filepath.Join(opts.AppPath, "x", opts.ModuleName, "genesis.go")

		dstHelper, err := astutils.NewDstHelper(path)
		defer dstHelper.Close()

		if err != nil {
			return err
		}

		templateModuleInit := `// Set all the %[1]v
for _, elem := range genState.%[2]vList {
	k.Set%[2]v(ctx, elem)
}

// Set %[1]v count
k.Set%[2]vCount(ctx, genState.%[2]vCount)`
		replacementModuleInit := fmt.Sprintf(
			templateModuleInit,
			opts.TypeName.LowerCamel,
			opts.TypeName.UpperCamel,
		)

		err = typed.AddToModuleInitGenesis(dstHelper, replacementModuleInit)
		if err != nil {
			return err
		}

		templateModuleExport := `genesis.%[1]vList = k.GetAll%[1]v(ctx)
genesis.%[1]vCount = k.Get%[1]vCount(ctx)`
		replacementModuleExport := fmt.Sprintf(
			templateModuleExport,
			opts.TypeName.UpperCamel,
		)

		err = typed.AddToModuleExportGenesis(dstHelper, replacementModuleExport)
		if err != nil {
			return err
		}

		content, err := dstHelper.Content()
		if err != nil {
			return err
		}

		newFile := genny.NewFileS(path, content)
		return r.File(newFile)
	}
}

func genesisTestsModify(opts *typed.Options) genny.RunFn {
	return func(r *genny.Runner) error {
		path := filepath.Join(opts.AppPath, "x", opts.ModuleName, "genesis_test.go")

		dstHelper, err := astutils.NewDstHelper(path)
		defer dstHelper.Close()

		if err != nil {
			return err
		}

		templateState := `%[1]vList: []types.%[1]v{
		{
			Id: 0,
		},
		{
			Id: 1,
		},
	},
	%[1]vCount: 2,`
		replacementValid := fmt.Sprintf(
			templateState,
			opts.TypeName.UpperCamel,
		)
		err = typed.AddToTestGenesisState(dstHelper, replacementValid)
		_ = replacementValid
		if err != nil {
			return err
		}

		templateAssert := `require.ElementsMatch(t, genesisState.%[1]vList, got.%[1]vList)
require.Equal(t, genesisState.%[1]vCount, got.%[1]vCount)`
		replacementTests := fmt.Sprintf(
			templateAssert,
			opts.TypeName.UpperCamel,
		)

		err = typed.AddToTestGenesisRequire(dstHelper, replacementTests)
		if err != nil {
			return err
		}

		content, err := dstHelper.Content()
		if err != nil {
			return err
		}

		newFile := genny.NewFileS(path, content)
		return r.File(newFile)
	}
}

func genesisTypesTestsModify(replacer placeholder.Replacer, opts *typed.Options) genny.RunFn {
	return func(r *genny.Runner) error {
		path := filepath.Join(opts.AppPath, "x", opts.ModuleName, "types/genesis_test.go")

		dstHelper, err := astutils.NewDstHelper(path)
		defer dstHelper.Close()

		if err != nil {
			return err
		}

		templateValid := `%[2]vList: []types.%[2]v{
	{
		Id: 0,
	},
	{
		Id: 1,
	},
},
%[2]vCount: 2,
%[1]v`
		replacementValid := fmt.Sprintf(
			templateValid,
			module.PlaceholderTypesGenesisValidField,
			opts.TypeName.UpperCamel,
		)
		// content := replacer.Replace(f.String(), module.PlaceholderTypesGenesisValidField, replacementValid)

		typed.AddToTypesTestGenesisState(dstHelper, replacementValid)
		templateTests := `{
	desc:     "duplicated %[2]v",
	genState: &types.GenesisState{
		%[3]vList: []types.%[3]v{
			{
				Id: 0,
			},
			{
				Id: 0,
			},
		},
	},
	valid:    false,
},
{
	desc:     "invalid %[2]v count",
	genState: &types.GenesisState{
		%[3]vList: []types.%[3]v{
			{
				Id: 1,
			},
		},
		%[3]vCount: 0,
	},
	valid:    false,
},
%[1]v`
		replacementTests := fmt.Sprintf(
			templateTests,
			module.PlaceholderTypesGenesisTestcase,
			opts.TypeName.LowerCamel,
			opts.TypeName.UpperCamel,
		)
		// content = replacer.Replace(content, module.PlaceholderTypesGenesisTestcase, replacementTests)
		typed.AddTestToGenesisStateValidate(dstHelper, replacementTests)

		content, err := dstHelper.Content()
		if err != nil {
			return err
		}

		newFile := genny.NewFileS(path, content)
		return r.File(newFile)
	}
}
