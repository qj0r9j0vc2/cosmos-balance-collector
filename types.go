package main

type BlockResponse struct {
	Result  BlockResult `json:"result"`
	ID      int64       `json:"id"`
	Jsonrpc string      `json:"jsonrpc"`
}

type BlockResult struct {
	Block HeaderResult `json:"block"`
}

type HeaderResult struct {
	Header Header `json:"header"`
}

type Header struct {
	ValidatorsHash     string      `json:"validators_hash"`
	ChainID            string      `json:"chain_id"`
	ConsensusHash      string      `json:"consensus_hash"`
	ProposerAddress    string      `json:"proposer_address"`
	NextValidatorsHash string      `json:"next_validators_hash"`
	Version            Version     `json:"version"`
	DataHash           string      `json:"data_hash"`
	LastResultsHash    string      `json:"last_results_hash"`
	LastBlockID        LastBlockID `json:"last_block_id"`
	EvidenceHash       string      `json:"evidence_hash"`
	AppHash            string      `json:"app_hash"`
	Time               string      `json:"time"`
	Height             string      `json:"height"`
	LastCommitHash     string      `json:"last_commit_hash"`
}

type LastBlockID struct {
	Parts Parts  `json:"parts"`
	Hash  string `json:"hash"`
}

type Parts struct {
	Total int64  `json:"total"`
	Hash  string `json:"hash"`
}

type Version struct {
	Block string `json:"block"`
}

type ABCIQueryResult struct {
	Result  *ABCIQueryResponse `json:"result"`
	ID      int64              `json:"id"`
	Jsonrpc string             `json:"jsonrpc"`
}

type ABCIQueryResponse struct {
	Response Response `json:"response"`
}

type Response struct {
	Code      int64  `json:"code"`
	Codespace string `json:"codespace"`
	Log       string `json:"log"`
	Index     string `json:"index"`
	Value     string `json:"value"`
	Info      string `json:"info"`
	Height    string `json:"height"`
}

type StatusResponse struct {
	Result  StatusResult `json:"result"`
	ID      int64        `json:"id"`
	Jsonrpc string       `json:"jsonrpc"`
}

type StatusResult struct {
	NodeInfo      NodeInfo      `json:"node_info"`
	ValidatorInfo ValidatorInfo `json:"validator_info"`
	SyncInfo      SyncInfo      `json:"sync_info"`
}

type NodeInfo struct {
	ProtocolVersion ProtocolVersion `json:"protocol_version"`
	Other           Other           `json:"other"`
	Channels        string          `json:"channels"`
	ListenAddr      string          `json:"listen_addr"`
	ID              string          `json:"id"`
	Moniker         string          `json:"moniker"`
	Version         string          `json:"version"`
	Network         string          `json:"network"`
}

type Other struct {
	TxIndex    string `json:"tx_index"`
	RPCAddress string `json:"rpc_address"`
}

type ProtocolVersion struct {
	App   string `json:"app"`
	Block string `json:"block"`
	P2P   string `json:"p2p"`
}

type SyncInfo struct {
	EarliestBlockHash   string `json:"earliest_block_hash"`
	LatestBlockTime     string `json:"latest_block_time"`
	EarliestBlockHeight string `json:"earliest_block_height"`
	LatestBlockHash     string `json:"latest_block_hash"`
	LatestAppHash       string `json:"latest_app_hash"`
	EarliestAppHash     string `json:"earliest_app_hash"`
	CatchingUp          bool   `json:"catching_up"`
	EarliestBlockTime   string `json:"earliest_block_time"`
	LatestBlockHeight   string `json:"latest_block_height"`
}

type ValidatorInfo struct {
	Address     string `json:"address"`
	PubKey      PubKey `json:"pub_key"`
	VotingPower string `json:"voting_power"`
}

type PubKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
