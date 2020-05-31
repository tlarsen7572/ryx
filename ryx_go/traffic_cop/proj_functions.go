package traffic_cop

import (
	"errors"
	"github.com/tlarsen7572/Golang-Public/ryx/ryxdoc"
	"github.com/tlarsen7572/Golang-Public/ryx/ryxnode"
	"github.com/tlarsen7572/Golang-Public/ryx/tool_data_loader"
	"path/filepath"
)

type NodeStructure struct {
	ToolId      int
	X           float64
	Y           float64
	Width       float64
	Height      float64
	Plugin      string
	StoredMacro string
	FoundMacro  string
	Category    string
}

type DocumentStructure struct {
	Nodes         []NodeStructure
	Connections   []*ryxdoc.RyxConn
	MacroToolData []tool_data_loader.ToolData
}

const getProjectStructureFunc = `GetProjectStructure`
const getDocumentStructureFunc = `GetDocumentStructure`
const whereUsedFunc = `WhereUsed`
const renameFilesFunc = `RenameFiles`
const moveFilesFunc = `MoveFiles`
const makeFilesAbsoluteFunc = `MakeFilesAbsolute`
const makeFilesRelativeFunc = `MakeFilesRelative`
const makeAllRelativeFunc = `MakeAllFilesRelative`
const makeAllAbsoluteFunc = `MakeAllFilesAbsolute`
const renameFolderFunc = `RenameFolder`
const listMacrosInProjectFunc = `ListMacrosInProject`
const batchUpdateMacroSettingsFunc = `BatchUpdateMacroSettings`
const invalidProjFunc = `invalid project function`

func handleProjFunction(call FunctionCall, data *TrafficCopData) FunctionResponse {
	switch call.Function {
	case getProjectStructureFunc:
		return getProjectStructure(data)
	case getDocumentStructureFunc:
		return getDocumentStructure(call, data)
	case whereUsedFunc:
		return whereUsed(call, data)
	case renameFilesFunc:
		return renameFiles(call, data)
	case moveFilesFunc:
		return moveFiles(call, data)
	case makeFilesAbsoluteFunc:
		return makeFilesAbsolute(call, data)
	case makeFilesRelativeFunc:
		return makeFilesRelative(call, data)
	case makeAllRelativeFunc:
		return makeAllRelative(data)
	case makeAllAbsoluteFunc:
		return makeAllAbsolute(data)
	case renameFolderFunc:
		return renameFolder(call, data)
	case listMacrosInProjectFunc:
		return listMacrosInProject(data)
	case batchUpdateMacroSettingsFunc:
		return batchUpdateMacroSettings(call, data)
	default:
		return _errorResponse(errors.New(invalidProjFunc))
	}
}

func getProjectStructure(data *TrafficCopData) FunctionResponse {
	structure, err := data.Project.Structure()
	if err != nil {
		return _errorResponse(err)
	}
	return _validResponse(structure)
}

func getDocumentStructure(call FunctionCall, data *TrafficCopData) FunctionResponse {
	var filePath string
	var ok bool
	if filePath, ok = call.Parameters[`FilePath`].(string); !ok {
		return _errorResponse(_stringParamErr(`FilePath`))
	}

	doc, err := data.Project.RetrieveDocument(filePath)
	if err != nil {
		return _errorResponse(err)
	}

	folderPath := filepath.Dir(filePath)
	macroPaths := append(data.MacroPaths, folderPath)

	nodes := []NodeStructure{}
	toolData := []tool_data_loader.ToolData{}
	for _, node := range doc.Nodes { // It is ok to use RyxDoc.Nodes here
		nodes, toolData = processDocStructureNode(call, node, macroPaths, nodes, toolData)
	}

	connections := doc.Connections
	if connections == nil {
		connections = []*ryxdoc.RyxConn{}
	}

	docStructure := DocumentStructure{
		Nodes:         nodes,
		Connections:   connections,
		MacroToolData: toolData,
	}
	return _validResponse(docStructure)
}

func processDocStructureNode(call FunctionCall, node *ryxnode.RyxNode, macroPaths []string, nodes []NodeStructure, toolData []tool_data_loader.ToolData) ([]NodeStructure, []tool_data_loader.ToolData) {
	id, err := node.ReadId()
	if err != nil {
		return nodes, toolData
	}
	plugin := node.ReadPlugin()
	if plugin == `AlteryxGuiToolkit.Questions.Tab.Tab` {
		return nodes, toolData
	}
	position, err := node.ReadPosition()
	if err != nil {
		position = ryxnode.Position{X: 0, Y: 0, Width: 0, Height: 0}
	}
	macro := node.ReadMacro(macroPaths...)
	category := node.ReadCategory().String()
	nodes = append(nodes, NodeStructure{
		ToolId:      id,
		X:           position.X,
		Y:           position.Y,
		Width:       position.Width,
		Height:      position.Height,
		Plugin:      plugin,
		StoredMacro: macro.StoredPath,
		FoundMacro:  macro.FoundPath,
		Category:    category,
	})
	if plugin := macro.StoredPath; plugin != `` && macro.FoundPath != `` {
		needsToolData := true
		for _, existing := range call.Config.ToolData {
			if plugin == existing.Plugin {
				needsToolData = false
				break
			}
		}
		if needsToolData {
			for _, alreadyGotIt := range toolData {
				if alreadyGotIt.Plugin == macro.FoundPath {
					needsToolData = false
					break
				}
			}
		}
		if needsToolData {
			data, err := tool_data_loader.ReadSingleMacro(macro.FoundPath, ``)
			if err == nil {
				toolData = append(toolData, data)
			}
		}
	}
	for _, childNode := range node.ChildNodes {
		nodes, toolData = processDocStructureNode(call, childNode, macroPaths, nodes, toolData)
	}
	return nodes, toolData
}

func whereUsed(call FunctionCall, data *TrafficCopData) FunctionResponse {
	path, ok := call.Parameters[`FilePath`].(string)
	if !ok {
		return _errorResponse(_stringParamErr(`FilePath`))
	}
	whereUsed := data.Project.WhereUsed(path)
	return _validResponse(whereUsed)
}

func makeFilesAbsolute(call FunctionCall, data *TrafficCopData) FunctionResponse {
	macros, err := _parseStringList(call.Parameters, `Files`)
	if err != nil {
		return _errorResponse(err)
	}
	result := data.Project.MakeFilesAbsolute(macros)
	return _validResponse(result)
}

func makeFilesRelative(call FunctionCall, data *TrafficCopData) FunctionResponse {
	macros, err := _parseStringList(call.Parameters, `Files`)
	if err != nil {
		return _errorResponse(err)
	}
	result := data.Project.MakeFilesRelative(macros)
	return _validResponse(result)
}

func makeAllRelative(data *TrafficCopData) FunctionResponse {
	result := data.Project.MakeAllFilesRelative()
	return _validResponse(result)
}

func makeAllAbsolute(data *TrafficCopData) FunctionResponse {
	result := data.Project.MakeAllFilesAbsolute()
	return _validResponse(result)
}

func renameFiles(call FunctionCall, data *TrafficCopData) FunctionResponse {
	fromFiles, err := _parseStringList(call.Parameters, `From`)
	if err != nil {
		return _errorResponse(err)
	}
	toFiles, err := _parseStringList(call.Parameters, `To`)
	if err != nil {
		return _errorResponse(err)
	}
	errFiles, err := data.Project.RenameFiles(fromFiles, toFiles)
	if err != nil {
		return _errorResponse(err)
	}
	return _validResponse(errFiles)
}

func moveFiles(call FunctionCall, data *TrafficCopData) FunctionResponse {
	fromFiles, err := _parseStringList(call.Parameters, `Files`)
	if err != nil {
		return _errorResponse(err)
	}
	to, ok := call.Parameters[`MoveTo`].(string)
	if !ok {
		return _errorResponse(_stringParamErr(`MoveTo`))
	}
	errFiles, err := data.Project.MoveFiles(fromFiles, to)
	if err != nil {
		_errorResponse(err)
	}
	return _validResponse(errFiles)
}

func renameFolder(call FunctionCall, data *TrafficCopData) FunctionResponse {
	from, ok := call.Parameters[`From`].(string)
	if !ok {
		return _errorResponse(_stringParamErr(`From`))
	}
	to, ok := call.Parameters[`To`].(string)
	if !ok {
		return _errorResponse(_stringParamErr(`To`))
	}
	err := data.Project.RenameFolder(from, to)
	return _errorResponse(err)
}

func listMacrosInProject(data *TrafficCopData) FunctionResponse {
	macros, err := data.Project.ListMacrosUsedInProject()
	if err != nil {
		return _errorResponse(err)
	}
	return _validResponse(macros)
}

func batchUpdateMacroSettings(call FunctionCall, data *TrafficCopData) FunctionResponse {
	name, ok := call.Parameters[`Name`].(string)
	if !ok {
		return _errorResponse(_stringParamErr(`Name`))
	}
	newSetting, ok := call.Parameters[`NewSetting`].(string)
	if !ok {
		return _errorResponse(_stringParamErr(`NewSetting`))
	}
	onlyFoundPaths, err := _parseStringList(call.Parameters, `OnlyFoundPaths`)
	if err != nil {
		return _errorResponse(err)
	}
	onlyStoredPaths, err := _parseStringList(call.Parameters, `OnlyStoredPaths`)
	if err != nil {
		return _errorResponse(err)
	}
	changed, err := data.Project.BatchChangeMacroSettings(name, newSetting, onlyFoundPaths, onlyStoredPaths)
	if err != nil {
		return _errorResponse(err)
	}
	return _validResponse(changed)
}
