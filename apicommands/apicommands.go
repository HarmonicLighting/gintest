package apicommands

type CommandType int
type commandName string

type commandMap map[CommandType]commandName

const (
	APIMessagesList CommandType = iota // The first iota is 0, this one is reserved
	ServerCompleteSignalList
	ServerSignalUpdateListPush
	ServerNConnectionsPush
	ServerSignalUpdatePush
)

var cmap commandMap

func init() {
	cmap = make(map[CommandType]commandName)
	cmap[APIMessagesList] = "APIMessageList"
	cmap[ServerCompleteSignalList] = "CompleteSignalList"
	cmap[ServerSignalUpdateListPush] = "SignalUpdateListPush"
	cmap[ServerNConnectionsPush] = "NConnectionsPush"
	cmap[ServerSignalUpdatePush] = "SignalUpdatePush"
}
