package filters

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	gethcmn "github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	gethfilters "github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/rpc"

	motypes "github.com/smartbch/moeingevm/types"
	mapi "github.com/smartbch/smartbch/api"
)

var _ PublicFilterAPI = (*filterAPI)(nil)

var (
	deadline = 5 * time.Minute // consider a filter inactive if it has not been polled for within deadline
)

type PublicFilterAPI interface {
	GetFilterChanges(id rpc.ID) (interface{}, error)
	GetFilterLogs(id rpc.ID) ([]*gethtypes.Log, error)
	GetLogs(crit gethfilters.FilterCriteria) ([]*gethtypes.Log, error)
	NewBlockFilter() rpc.ID
	NewFilter(crit gethfilters.FilterCriteria) (rpc.ID, error)
	UninstallFilter(id rpc.ID) bool
}

type filterAPI struct {
	backend   mapi.BackendService
	events    *EventSystem
	filtersMu sync.Mutex
	filters   map[rpc.ID]*filter
}

// filter is a helper struct that holds meta information over the filter type
// and associated subscription in the event system.
type filter struct {
	typ      Type
	deadline *time.Timer // filter is inactive when deadline triggers
	hashes   []gethcmn.Hash
	crit     gethfilters.FilterCriteria
	logs     []*gethtypes.Log
	s        *Subscription // associated subscription in event system
}

func NewAPI(backend mapi.BackendService) PublicFilterAPI {
	_api := &filterAPI{
		backend: backend,
		filters: make(map[rpc.ID]*filter),
		events:  NewEventSystem(backend, false),
	}

	go _api.timeoutLoop()
	return _api
}

// timeoutLoop runs every 5 minutes and deletes filters that have not been recently used.
// Tt is started when the api is created.
func (api *filterAPI) timeoutLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		<-ticker.C
		api.filtersMu.Lock()
		for id, f := range api.filters {
			select {
			case <-f.deadline.C:
				f.s.Unsubscribe()
				delete(api.filters, id)
			default:
				continue
			}
		}
		api.filtersMu.Unlock()
	}
}

// NewFilter creates a new filter and returns the filter id. It can be
// used to retrieve logs when the state changes. This method cannot be
// used to fetch logs that are already stored in the state.
//
// Default criteria for the from and to block are "latest".
// Using "latest" as block number will return logs for mined blocks.
// Using "pending" as block number returns logs for not yet mined (pending) blocks.
// In case logs are removed (chain reorg) previously returned logs are returned
// again but with the removed property set to true.
//
// In case "fromBlock" > "toBlock" an error is returned.
//
// https://eth.wiki/json-rpc/API#eth_newFilter
func (api *filterAPI) NewFilter(crit gethfilters.FilterCriteria) (filterID rpc.ID, err error) {
	logs := make(chan []*gethtypes.Log)
	logsSub, err := api.events.SubscribeLogs(ethereum.FilterQuery(crit), logs)
	if err != nil {
		return "", err
	}

	api.filtersMu.Lock()
	api.filters[logsSub.ID] = &filter{
		typ:      LogsSubscription,
		crit:     crit,
		deadline: time.NewTimer(deadline),
		logs:     make([]*gethtypes.Log, 0),
		s:        logsSub,
	}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case l := <-logs:
				api.filtersMu.Lock()
				if f, found := api.filters[logsSub.ID]; found {
					f.logs = append(f.logs, l...)
				}
				api.filtersMu.Unlock()
			case <-logsSub.Err():
				api.filtersMu.Lock()
				delete(api.filters, logsSub.ID)
				api.filtersMu.Unlock()
				return
			}
		}
	}()

	return logsSub.ID, nil
}

// NewBlockFilter creates a filter that fetches blocks that are imported into the chain.
// It is part of the filter package since polling goes with eth_getFilterChanges.
//
// https://eth.wiki/json-rpc/API#eth_newblockfilter
func (api *filterAPI) NewBlockFilter() rpc.ID {
	var (
		headers   = make(chan *motypes.Header)
		headerSub = api.events.SubscribeNewHeads(headers)
	)

	api.filtersMu.Lock()
	api.filters[headerSub.ID] = &filter{
		typ:      BlocksSubscription,
		deadline: time.NewTimer(deadline),
		hashes:   make([]gethcmn.Hash, 0),
		s:        headerSub,
	}
	api.filtersMu.Unlock()

	go func() {
		for {
			select {
			case h := <-headers:
				api.filtersMu.Lock()
				if f, found := api.filters[headerSub.ID]; found {
					f.hashes = append(f.hashes, h.Hash())
				}
				api.filtersMu.Unlock()
			case <-headerSub.Err():
				api.filtersMu.Lock()
				delete(api.filters, headerSub.ID)
				api.filtersMu.Unlock()
				return
			}
		}
	}()

	return headerSub.ID
}

// UninstallFilter removes the filter with the given filter id.
//
// https://eth.wiki/json-rpc/API#eth_uninstallfilter
func (api *filterAPI) UninstallFilter(id rpc.ID) bool {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	if found {
		delete(api.filters, id)
	}
	api.filtersMu.Unlock()
	if found {
		f.s.Unsubscribe()
	}

	return found
}

// GetFilterChanges returns the logs for the filter with the given id since
// last time it was called. This can be used for polling.
//
// For pending transaction and block filters the result is []common.Hash.
// (pending)Log filters return []Log.
//
// https://eth.wiki/json-rpc/API#eth_getfilterchanges
func (api *filterAPI) GetFilterChanges(id rpc.ID) (interface{}, error) {
	api.filtersMu.Lock()
	defer api.filtersMu.Unlock()

	f, found := api.filters[id]
	if !found {
		return nil, fmt.Errorf("filter %s not found", id)
	}

	if !f.deadline.Stop() {
		// timer expired but filter is not yet removed in timeout loop
		// receive timer value and reset timer
		<-f.deadline.C
	}
	f.deadline.Reset(deadline)

	switch f.typ {
	case /*PendingTransactionsSubscription, */ BlocksSubscription:
		hashes := f.hashes
		f.hashes = nil
		return returnHashes(hashes), nil
	case LogsSubscription /*, MinedAndPendingLogsSubscription*/ :
		logs := make([]*gethtypes.Log, len(f.logs))
		copy(logs, f.logs)
		f.logs = []*gethtypes.Log{}
		return returnLogs(logs), nil
	default:
		return nil, fmt.Errorf("invalid filter %s type %d", id, f.typ)
	}
}

// GetFilterLogs returns the logs for the filter with the given id.
// If the filter could not be found an empty array of logs is returned.
//
// https://eth.wiki/json-rpc/API#eth_getfilterlogs
func (api *filterAPI) GetFilterLogs(id rpc.ID) ([]*gethtypes.Log, error) {
	api.filtersMu.Lock()
	f, found := api.filters[id]
	api.filtersMu.Unlock()

	if !found || f.typ != LogsSubscription {
		return nil, fmt.Errorf("filter not found")
	}
	return api.GetLogs(f.crit)
}

// GetLogs returns logs matching the given argument that are stored within the state.
//
// https://eth.wiki/json-rpc/API#eth_getLogs
func (api *filterAPI) GetLogs(crit gethfilters.FilterCriteria) ([]*gethtypes.Log, error) {
	if crit.BlockHash != nil {
		// Block filter requested, construct a single-shot filter
		filter := NewBlockFilter(api.backend, *crit.BlockHash, crit.Addresses, crit.Topics)

		// Run the filter and return all the logs
		logs, err := filter.Logs(context.TODO())
		if err != nil {
			return nil, err
		}
		return returnLogs(logs), nil
	}

	// Convert the RPC block numbers into internal representations
	begin := rpc.LatestBlockNumber.Int64()
	if crit.FromBlock != nil {
		begin = crit.FromBlock.Int64()
	}
	end := rpc.LatestBlockNumber.Int64()
	if crit.ToBlock != nil {
		end = crit.ToBlock.Int64()
	}
	if begin < 0 {
		begin = api.backend.LatestHeight()
	}
	if end < 0 {
		end = api.backend.LatestHeight()
	}

	logs, err := api.backend.QueryLogs(crit.Addresses, crit.Topics, uint32(begin), uint32(end+1), filterFunc)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("Why? begin %d end %d logs %#v\n", begin, end, logs)

	return motypes.ToGethLogs(logs), nil
}

// returnHashes is a helper that will return an empty hash array case the given hash array is nil,
// otherwise the given hashes array is returned.
func returnHashes(hashes []gethcmn.Hash) []gethcmn.Hash {
	if hashes == nil {
		return []gethcmn.Hash{}
	}
	return hashes
}

// returnLogs is a helper that will return an empty log array in case the given logs array is nil,
// otherwise the given logs array is returned.
func returnLogs(logs []*gethtypes.Log) []*gethtypes.Log {
	if logs == nil {
		return []*gethtypes.Log{}
	}
	return logs
}
