package sudo

/*
Wasm contracts have the special entrypoint called sudo. The main purpose of the entrypoint is to be called from a trusted cosmos module, e.g. via a governance process.
We use the entrypoint to send back an ibc acknowledgement for an ibc transaction.
The package contains the code to postprocess incoming from a relayer acknowledgement and pass it to the  ibc transaction contract initiator
*/

import (
	"encoding/json"
	"fmt"

	"github.com/CosmWasm/wasmd/x/wasm"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	"github.com/tendermint/tendermint/libs/log"
)

// SudoMessageTxQueryResult is the messages that gets passed to a contract's
// sudo handler when a tx was submitted for a tx query.
type SudoMessageTxQueryResult struct {
	TxQueryResult struct {
		QueryID uint64 `json:"query_id"`
		Height  uint64 `json:"height"`
		Data    []byte `json:"data"`
	} `json:"tx_query_result"`
}

type SudoMessageKVQueryResult struct {
	KVQueryResult struct {
		QueryID uint64 `json:"query_id"`
	} `json:"kv_query_result"`
}

type SudoMessageTimeout struct {
	Timeout struct {
		Request channeltypes.Packet `json:"request"`
	} `json:"timeout"`
}

type SudoMessageResponse struct {
	Response struct {
		Request channeltypes.Packet `json:"request"`
		Data    []byte              `json:"data"` // Message data
	} `json:"response"`
}

type SudoMessageError struct {
	Error struct {
		Request channeltypes.Packet `json:"request"`
		Details string              `json:"details"`
	} `json:"error"`
}

type SudoMessageOpenAck struct {
	OpenAck OpenAckDetails `json:"open_ack"`
}

type OpenAckDetails struct {
	PortID                string `json:"port_id"`
	ChannelID             string `json:"channel_id"`
	CounterpartyChannelId string `json:"counterparty_channel_id"`
	CounterpartyVersion   string `json:"counterparty_version"`
}

type SudoHandler struct {
	moduleName string
	wasmKeeper *wasm.Keeper
}

func (s *SudoHandler) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", s.moduleName))
}

func NewSudoHandler(wasmKeeper *wasm.Keeper, moduleName string) SudoHandler {
	return SudoHandler{
		moduleName: moduleName,
		wasmKeeper: wasmKeeper,
	}
}

func (s *SudoHandler) SudoResponse(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	request channeltypes.Packet,
	msg []byte,
) ([]byte, error) {
	s.Logger(ctx).Debug("SudoResponse", "contractAddress", contractAddress, "request", request, "msg", msg)

	// TODO: basically just for unit tests right now. But i think we will have the same logic in the production
	if !s.wasmKeeper.HasContractInfo(ctx, contractAddress) {
		return nil, nil
	}

	x := SudoMessageResponse{}
	x.Response.Data = msg
	x.Response.Request = request
	m, err := json.Marshal(x)
	if err != nil {
		s.Logger(ctx).Error("SudoResponse: failed to marshal SudoMessageResponse message",
			"error", err, "request", request, "contract_address", contractAddress)
		return nil, sdkerrors.Wrap(err, "failed to marshal SudoMessageResponse message")
	}

	resp, err := s.wasmKeeper.Sudo(ctx, contractAddress, m)
	if err != nil {
		s.Logger(ctx).Debug("SudoResponse: failed to Sudo",
			"error", err, "contract_address", contractAddress)
		return nil, err
	}

	return resp, nil
}

func (s *SudoHandler) SudoTimeout(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	request channeltypes.Packet,
) ([]byte, error) {
	s.Logger(ctx).Info("SudoTimeout", "contractAddress", contractAddress, "request", request)

	// TODO: basically just for unit tests right now. But i think we will have the same logic in the production
	if !s.wasmKeeper.HasContractInfo(ctx, contractAddress) {
		return nil, nil
	}

	x := SudoMessageTimeout{}
	x.Timeout.Request = request
	m, err := json.Marshal(x)
	if err != nil {
		s.Logger(ctx).Error("failed to marshal SudoMessageTimeout message",
			"error", err, "request", request, "contract_address", contractAddress)
		return nil, sdkerrors.Wrap(err, "failed to marshal SudoMessageTimeout message")
	}

	s.Logger(ctx).Info("SudoTimeout sending request", "data", string(m))

	resp, err := s.wasmKeeper.Sudo(ctx, contractAddress, m)
	if err != nil {
		s.Logger(ctx).Debug("SudoTimeout: failed to Sudo",
			"error", err, "contract_address", contractAddress)
		return nil, err
	}

	return resp, nil
}

func (s *SudoHandler) SudoError(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	request channeltypes.Packet,
	details string,
) ([]byte, error) {
	s.Logger(ctx).Debug("SudoError", "contractAddress", contractAddress, "request", request)

	// TODO: basically just for unit tests right now. But i think we will have the same logic in the production
	if !s.wasmKeeper.HasContractInfo(ctx, contractAddress) {
		return nil, nil
	}

	x := SudoMessageError{}
	x.Error.Request = request
	x.Error.Details = details
	m, err := json.Marshal(x)
	if err != nil {
		s.Logger(ctx).Error("SudoError: failed to marshal SudoMessageError message",
			"error", err, "contract_address", contractAddress, "request", request)
		return nil, sdkerrors.Wrap(err, "failed to marshal SudoMessageError message")
	}

	resp, err := s.wasmKeeper.Sudo(ctx, contractAddress, m)
	if err != nil {
		s.Logger(ctx).Debug("SudoError: failed to Sudo",
			"error", err, "contract_address", contractAddress)
		return nil, err
	}

	return resp, nil
}

func (s *SudoHandler) SudoOpenAck(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	details OpenAckDetails,
) ([]byte, error) {
	s.Logger(ctx).Debug("SudoOpenAck", "contractAddress", contractAddress)

	// TODO: basically just for unit tests right now. But i think we will have the same logic in the production
	if !s.wasmKeeper.HasContractInfo(ctx, contractAddress) {
		return nil, nil
	}

	x := SudoMessageOpenAck{}
	x.OpenAck = details
	m, err := json.Marshal(x)
	if err != nil {
		s.Logger(ctx).Error("SudoOpenAck: failed to marshal SudoMessageOpenAck message",
			"error", err, "contract_address", contractAddress)
		return nil, sdkerrors.Wrap(err, "failed to marshal SudoMessageOpenAck message")
	}
	s.Logger(ctx).Info("SudoOpenAck sending request", "data", string(m))

	resp, err := s.wasmKeeper.Sudo(ctx, contractAddress, m)
	if err != nil {
		s.Logger(ctx).Debug("SudoOpenAck: failed to Sudo",
			"error", err, "contract_address", contractAddress)
		return nil, err
	}

	return resp, nil
}

// SudoTxQueryResult is used to pass a tx query result to the contract that registered the query
// to:
// 		1. check whether the transaction actually satisfies the initial query arguments;
// 		2. execute business logic related to the tx query result / save the result to state.
func (s *SudoHandler) SudoTxQueryResult(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	queryID uint64,
	height int64,
	data []byte,
) ([]byte, error) {
	s.Logger(ctx).Debug("SudoTxQueryResult", "contractAddress", contractAddress)

	// TODO: basically just for unit tests right now. But i think we will have the same logic in the production
	if !s.wasmKeeper.HasContractInfo(ctx, contractAddress) {
		s.Logger(ctx).Debug("SudoTxQueryResult: contract not found", "contractAddress", contractAddress)
		return nil, nil
	}

	x := SudoMessageTxQueryResult{}
	x.TxQueryResult.QueryID = queryID
	x.TxQueryResult.Height = uint64(height)
	x.TxQueryResult.Data = data

	m, err := json.Marshal(x)
	if err != nil {
		s.Logger(ctx).Error("failed to marshal SudoMessageTxQueryResult message",
			"error", err, "contract_address", contractAddress)
		return nil, sdkerrors.Wrap(err, "failed to marshal SudoMessageTxQueryResult message")
	}

	resp, err := s.wasmKeeper.Sudo(ctx, contractAddress, m)
	if err != nil {
		s.Logger(ctx).Debug("SudoTxQueryResult: failed to Sudo",
			"error", err, "contract_address", contractAddress)
		return nil, err
	}

	return resp, nil
}

// SudoKVQueryResult is used to pass a kv query id to the contract that registered the query
// when a query result is provided by the relayer.
func (s *SudoHandler) SudoKVQueryResult(
	ctx sdk.Context,
	contractAddress sdk.AccAddress,
	queryID uint64,
) ([]byte, error) {
	s.Logger(ctx).Info("SudoKVQueryResult", "contractAddress", contractAddress)

	// TODO: basically just for unit tests right now. But i think we will have the same logic in the production
	if !s.wasmKeeper.HasContractInfo(ctx, contractAddress) {
		s.Logger(ctx).Debug("contract was not found", "contractAddress", contractAddress)
		return nil, nil
	}

	x := SudoMessageKVQueryResult{}
	x.KVQueryResult.QueryID = queryID

	m, err := json.Marshal(x)
	if err != nil {
		s.Logger(ctx).Error("SudoKVQueryResult: failed to marshal SudoMessageKVQueryResult message",
			"error", err, "contract_address", contractAddress)
		return nil, sdkerrors.Wrap(err, "failed to marshal SudoMessageKVQueryResult message")
	}

	resp, err := s.wasmKeeper.Sudo(ctx, contractAddress, m)
	if err != nil {
		s.Logger(ctx).Debug("SudoKVQueryResult: failed to Sudo",
			"error", err, "contract_address", contractAddress)
		return nil, err
	}

	return resp, nil
}