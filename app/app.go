package app

import (
	"strings"

	"github.com/tendermint/basecoin/state"
	"github.com/tendermint/basecoin/types"
	. "github.com/tendermint/go-common"
	"github.com/tendermint/go-wire"
	"github.com/tendermint/governmint/gov"
	eyes "github.com/tendermint/merkleeyes/client"
	tmsp "github.com/tendermint/tmsp/types"
)

const (
	version   = "0.1"
	maxTxSize = 10240

	PluginTypeByteBase = 0x01
	PluginTypeByteEyes = 0x02
	PluginTypeByteGov  = 0x03

	PluginNameBase = "base"
	PluginNameEyes = "eyes"
	PluginNameGov  = "gov"
)

type Basecoin struct {
	eyesCli *eyes.Client
	govMint *gov.Governmint
	state   *state.State
	plugins *types.Plugins
}

func NewBasecoin(eyesCli *eyes.Client) *Basecoin {
	govMint := gov.NewGovernmint(eyesCli)
	state_ := state.NewState(eyesCli)
	plugins := types.NewPlugins()
	plugins.RegisterPlugin(PluginTypeByteGov, PluginNameGov, govMint)
	return &Basecoin{
		eyesCli: eyesCli,
		govMint: govMint,
		state:   state_,
		plugins: plugins,
	}
}

// TMSP::Info
func (app *Basecoin) Info() string {
	return Fmt("Basecoin v%v", version)
}

// TMSP::SetOption
func (app *Basecoin) SetOption(key string, value string) (log string) {
	PluginName, key := splitKey(key)
	if PluginName != PluginNameBase {
		// Set option on plugin
		plugin := app.plugins.GetByName(PluginName)
		if plugin == nil {
			return "Invalid plugin name: " + PluginName
		}
		return plugin.SetOption(key, value)
	} else {
		// Set option on basecoin
		switch key {
		case "chainID":
			app.state.SetChainID(value)
			return "Success"
		case "account":
			var err error
			var acc *types.Account
			wire.ReadJSONPtr(&acc, []byte(value), &err)
			if err != nil {
				return "Error decoding acc message: " + err.Error()
			}
			app.state.SetAccount(acc.PubKey.Address(), acc)
			return "Success"
		}
		return "Unrecognized option key " + key
	}
}

// TMSP::AppendTx
func (app *Basecoin) AppendTx(txBytes []byte) (res tmsp.Result) {
	if len(txBytes) > maxTxSize {
		return tmsp.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
	}
	// Decode tx
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return tmsp.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}
	// Validate and exec tx
	res = state.ExecTx(app.state, app.plugins, tx, false, nil)
	if res.IsErr() {
		return res.PrependLog("Error in AppendTx")
	}
	return tmsp.OK
}

// TMSP::CheckTx
func (app *Basecoin) CheckTx(txBytes []byte) (res tmsp.Result) {
	if len(txBytes) > maxTxSize {
		return tmsp.ErrBaseEncodingError.AppendLog("Tx size exceeds maximum")
	}
	// Decode tx
	var tx types.Tx
	err := wire.ReadBinaryBytes(txBytes, &tx)
	if err != nil {
		return tmsp.ErrBaseEncodingError.AppendLog("Error decoding tx: " + err.Error())
	}
	// Validate tx
	res = state.ExecTx(app.state, app.plugins, tx, true, nil)
	if res.IsErr() {
		return res.PrependLog("Error in CheckTx")
	}
	return tmsp.OK
}

// TMSP::Query
func (app *Basecoin) Query(query []byte) (res tmsp.Result) {
	if len(query) == 0 {
		return tmsp.ErrEncodingError.SetLog("Query cannot be zero length")
	}
	typeByte := query[0]
	query = query[1:]
	switch typeByte {
	case PluginTypeByteBase:
		return tmsp.OK.SetLog("This type of query not yet supported")
	case PluginTypeByteEyes:
		return app.eyesCli.QuerySync(query)
	case PluginTypeByteGov:
		return app.govMint.Query(query)
	}
	return tmsp.ErrBaseUnknownPlugin.SetLog(
		Fmt("Unknown plugin with type byte %X", typeByte))
}

// TMSP::Commit
func (app *Basecoin) Commit() (res tmsp.Result) {
	// First, commit all the plugins
	for _, plugin := range app.plugins.GetList() {
		res = plugin.Commit()
		if res.IsErr() {
			PanicSanity(Fmt("Error committing plugin %v", plugin.Name))
		}
	}
	// Then, commit eyes.
	res = app.eyesCli.CommitSync()
	if res.IsErr() {
		PanicSanity("Error getting hash: " + res.Error())
	}
	return res
}

// TMSP::InitChain
func (app *Basecoin) InitChain(validators []*tmsp.Validator) {
	for _, plugin := range app.plugins.GetList() {
		if _, ok := plugin.Plugin.(tmsp.BlockchainAware); ok {
			plugin.Plugin.(tmsp.BlockchainAware).InitChain(validators)
		}
	}
}

// TMSP::BeginBlock
func (app *Basecoin) BeginBlock(height uint64) {
	app.state.ResetCacheState()
	for _, plugin := range app.plugins.GetList() {
		if _, ok := plugin.Plugin.(tmsp.BlockchainAware); ok {
			plugin.Plugin.(tmsp.BlockchainAware).BeginBlock(height)
		}
	}
}

// TMSP::EndBlock
func (app *Basecoin) EndBlock(height uint64) (vals []*tmsp.Validator) {
	for _, plugin := range app.plugins.GetList() {
		if plugin.Plugin == app.govMint {
			vals = plugin.Plugin.(tmsp.BlockchainAware).EndBlock(height)
		} else {
			if _, ok := plugin.Plugin.(tmsp.BlockchainAware); ok {
				plugin.Plugin.(tmsp.BlockchainAware).EndBlock(height)
			}
		}
	}
	return
}

//----------------------------------------

// Splits the string at the first '/'.
// if there are none, the second string is nil.
func splitKey(key string) (prefix string, suffix string) {
	if strings.Contains(key, "/") {
		keyParts := strings.SplitN(key, "/", 2)
		return keyParts[0], keyParts[1]
	}
	return key, ""
}
