package store

import (
	"time"
	"bytes"
	"os"
	"io/ioutil"
	"log"
	"encoding/hex"
	"encoding/json"
	"crypto/sha256"
	"fmt"
	"io"
	"github.com/ortutay/decloud/conf"
	"github.com/ortutay/decloud/msg"
	"github.com/ortutay/decloud/util"
	"github.com/ortutay/decloud/rep"
)

const (

	SERVICE_NAME = "store"

	// TODO(ortutay): quote is not really a method of the service; may want to
	// put it in a separate package
	// put [size] [time]
	QUOTE_METHOD = "quote"

	// no arguments
	// returns [container-id] for the node
	ALLOC_METHOD = "alloc"

	// [container-id] [blob-id] optional body: [block-list]
	// returns {ok|error|block-list-request}
	PUT_METHOD = "put"

	// [container-id] [blob-id]
	GET_METHOD = "get"

	// [blob-id] [block-indexs] [salt]
	HASH_METHOD = "hash"
)

const BYTES_PER_BLOCK = 4096

// TODO(ortutay): these should be configured via flags
const MAX_BLOB_BYTES = 50 * 1e6 // 50 MB
const MAX_CONTAINER_BYTES = 500 * 1e6 // 500 MB

type BlockID string

func (b BlockID) String() string {
	return string(b)
}

type Block struct {
	ID BlockID
	Data []byte
}

func blockPath(id BlockID) string {
 	dir := util.ServiceDir(SERVICE_NAME) + "/blocks"
	err := util.MakeDir(dir)
	util.Ferr(err)
	return dir + "/" + id.String()
}

func NewBlock(data []byte) (*Block, error) {
	if len(data) > BYTES_PER_BLOCK {
		return nil, fmt.Errorf("block with size %v exceeds max %v",
			len(data), BYTES_PER_BLOCK)
	}
	id := util.Sha256AsString(data)
	return &Block{ID: BlockID(id), Data: data}, nil
}

type BlobID string

func (b BlobID) String() string {
	return string(b)
}

type Blob struct {
	ID BlobID
	Blocks []*Block
}

func NewBlob(blocks []*Block) (*Blob, error) {
	h := sha256.New()
	for _, block := range blocks {
		_, err := h.Write([]byte(block.Data))
		util.Ferr(err)
	}
	b := h.Sum([]byte{})
	id := hex.EncodeToString(b)
	blob := Blob{
		ID: BlobID(id),
		Blocks: blocks,
	}
	return &blob, nil
}

func NewBlobFromReader(r io.Reader) (*Blob, error) {
	blocks := make([]*Block, 0)
	for {
		buf := make([]byte, BYTES_PER_BLOCK)
		n, err := r.Read(buf)
		if err != nil && n == 0 {
			break
		}
		if err != nil {
			return nil, err
		}
		block, err := NewBlock(buf[:n])
		util.Ferr(err)
		util.Ferr(err)
		blocks = append(blocks, block)
	}
	return NewBlob(blocks)
}

func NewBlobFromDisk(id BlobID) (*Blob, error) {
	d := util.GetOrCreateDB(blobToBlocksDB())
	idsSer, err := d.Read(id.String())
	if idsSer == nil || len(idsSer) == 0 {
		return nil, fmt.Errorf("not found")
	}
	var ids []BlockID
	err = json.Unmarshal(idsSer, &ids)
	util.Ferr(err)
	var blocks []*Block
	for _, id := range ids {
		block, err := func () (*Block, error) {
			f, err := os.Open(blockPath(id))
			defer f.Close()
			if err != nil {
				return nil, err
			}
			data, err := ioutil.ReadAll(f)
			if len(data) > BYTES_PER_BLOCK {
				log.Fatalf("too big block %v of size %v", id, len(data))
			}
			block, err := NewBlock(data)
			util.Ferr(err)
			if block.ID != id {
				log.Fatalf("mismatched id's %v != %v", id, block.ID)
			}
			return block, nil
		}()
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return NewBlob(blocks)
}

func (b *Blob) String() string {
	s := fmt.Sprintf("%v", b.ID)
	for _, block := range b.Blocks {
		s += fmt.Sprintf(" [%v]", string(block.Data))
	}
	return s
}

func (b *Blob) ShortString() string {
	s := fmt.Sprintf("%v", b.ID[:8])
	for _, block := range b.Blocks {
		s += fmt.Sprintf(" [%v...]", string(block.Data[:8]))
	}
	return s
}

func (b *Blob) BlockIDs() []BlockID {
	ids := make([]BlockID, len(b.Blocks))
	for i, block := range b.Blocks {
		ids[i] = block.ID
	}
	return ids
}

type ContainerID string

func (c ContainerID) String() string {
	return string(c)
}

type Container struct {
	ID ContainerID `json:"id"`
	OwnerID msg.OcID `json:"ownerId"`
	BlobIDs []BlobID `json:"blobIds"`
}

func NewContainerFromDisk(id msg.OcID) *Container {
	d := util.GetOrCreateDB(containersDB())
	containerID := ocIDToContainerID(id)
	ser, _ := d.Read(id.String())
	if ser == nil || len(ser) == 0 {
		return &Container{ID: containerID, OwnerID: id}
	} else {
		var container Container
		err := json.Unmarshal(ser, &container)
		util.Ferr(err)
		return &container
	}
}

func (c *Container) WriteNewBlobID(id BlobID) {
	d := util.GetOrCreateDB(containersDB())
	c.BlobIDs = append(c.BlobIDs, id)
	blobIDsSer, err := json.Marshal(c)
	util.Ferr(err)
	err = d.Write(c.OwnerID.String(), blobIDsSer)
	util.Ferr(err)
}

func (c *Container) HasBlobID(targetID BlobID) bool {
	for _, id := range c.BlobIDs {
		if id == targetID {
			return true
		}
	}
	return false
}

func containersDB() string {
 	return util.ServiceDir(SERVICE_NAME) + "/containers.db"
}

type WorkPut struct {
	Blocks  int `json:"blocks"`
	Seconds int `json:"seconds"`
}

type WorkGet struct {
	Blocks int `json:"blocks"`
}

type WorkHash struct {
	Blocks int `json:"blocks"`
}

func MeasurePut(req *msg.OcReq) (*WorkPut, error) {
	return nil, nil
}

func MeasureGet(req *msg.OcReq) (*WorkGet, error) {
	return nil, nil
}

func MeasureHash(req *msg.OcReq) (*WorkHash, error) {
	return nil, nil
}

func (wp *WorkPut) Quote() *msg.PaymentValue {
	return nil
}

func (wp *WorkGet) Quote() *msg.PaymentValue {
	return nil
}

func (wp *WorkHash) Quote() *msg.PaymentValue {
	return nil
}

func NewPutReq() *msg.OcReq {
	return nil
}

func NewGetReq() *msg.OcReq {
	return nil
}

func NewHashReq() *msg.OcReq {
	return nil
}

type StoreService struct {
	Conf *conf.Conf
	lastWake int64
}

func (ss *StoreService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	println(fmt.Sprintf("store got request: %v", req))
	if req.Service != SERVICE_NAME {
		panic(fmt.Sprintf("unexpected service %s", req.Service))
	}

	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[ALLOC_METHOD] = ss.alloc
	methods[PUT_METHOD] = ss.put
	methods[GET_METHOD] = ss.get

	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return msg.NewRespError(msg.METHOD_UNSUPPORTED), nil
	}
}

func costForBytesSeconds(bytes int, seconds int) *msg.PaymentValue {
	costBtc := float64(bytes) * float64(seconds) * .000001
	return &msg.PaymentValue{Amount: util.B2S(costBtc), Currency: msg.BTC}
}

func (ss *StoreService) PeriodicWake() {
	now := time.Now().Unix()
	if ss.lastWake == 0 {
		ss.lastWake = now
	}
	period := int64(10)
	if now - ss.lastWake < period {
		return
	}
	ss.lastWake = now
	d := util.GetOrCreateDB(containersDB())
	keys := d.Keys()
	for {
		key := <-keys
		if len(key) == 0 {
			break
		}
		bytesUsed := 0
		id := msg.OcID(key)
		container := NewContainerFromDisk(id)
		seenBlocks := make(map[string]bool)
		for _, blobID := range container.BlobIDs {
			// TODO(ortutay): don't read blocks from disk just to find sizes
			blob, err := NewBlobFromDisk(blobID)
			if err != nil {
				continue
			}
			for _, block := range blob.Blocks {
				if _, ok := seenBlocks[block.ID.String()]; ok {
					continue
				}
				seenBlocks[block.ID.String()] = true
				bytesUsed += len(block.Data)
			}
		}
		costPv := costForBytesSeconds(bytesUsed, int(period))
		fmt.Printf("bytes %v used by %v..., cost += %f %v\n",
			bytesUsed, id.String()[:8], util.S2B(costPv.Amount), costPv.Currency)
		rec := rep.Record{
			Role: rep.SERVER,
			Service: SERVICE_NAME,
			Method: PUT_METHOD,
			Timestamp: int(now),
			ID: id,
			Status: rep.SUCCESS_UNPAID,
			PaymentValue: costPv,
			Perf: nil,
		}
		rep.Put(&rec)
	}
}

func (ss *StoreService) quote(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}

func ocIDToContainerID(id msg.OcID) ContainerID {
	return ContainerID(util.Sha256AsString([]byte(id.String())))
}

func (ss *StoreService) alloc(req *msg.OcReq) (*msg.OcResp, error) {
	// TODO(ortutay): may want multiple contianers per client
	id := ocIDToContainerID(req.ID)
	return msg.NewRespOk([]byte(id.String())), nil
}

func blobToBlocksDB() string {
 	return util.ServiceDir(SERVICE_NAME) + "/blob-to-blocks.db"
}

func storeBlob(blob *Blob) error {
	// TODO(ortutay): in the case of a failed write, we need to garbage collect
	for _, block := range blob.Blocks {
		path := blockPath(block.ID)
		if _, err := os.Stat(path); err == nil {
			continue
		}
		func () {
			f, err := os.Create(path)
			defer f.Close()
			util.Ferr(err)
			_, err = f.Write(block.Data)
			util.Ferr(err)
		}()
	}

	// Write blob -> blocks mapping
	ids := blob.BlockIDs()
	d := util.GetOrCreateDB(blobToBlocksDB())
	ser, err := json.Marshal(ids)
	util.Ferr(err)
	err = d.Write(blob.ID.String(), ser)
	util.Ferr(err)
	return nil
}

func updateIndexes(cont *Container) error {
	fmt.Printf("updateIndexes\n")
	return nil
}

func (ss *StoreService) put(req *msg.OcReq) (*msg.OcResp, error) {
	var containerID ContainerID
	var blobID BlobID
	if len(req.Args) == 0 {
		containerID = ocIDToContainerID(req.ID)
		// blob will be read from request
	} else if len(req.Args) == 1 {
		if req.Args[0] == "." {
			containerID = ocIDToContainerID(req.ID)
		} else {
			containerID = ContainerID(req.Args[0])
		}
		// blob will be read from request
	} else if len(req.Args) == 2 {
		if req.Args[0] == "." {
			containerID = ocIDToContainerID(req.ID)
		} else {
			containerID = ContainerID(req.Args[0])
		}
		blobID = BlobID(req.Args[1])
	} else {
		return msg.NewRespError(msg.INVALID_ARGUMENTS), nil
	}

	if containerID != ocIDToContainerID(req.ID) {
		resp := msg.NewRespErrorWithBody(msg.INVALID_ARGUMENTS,
			[]byte("Cannot access that container"))
		return resp, nil
	}

	fmt.Printf("put request for: %v %v\n", containerID, blobID)

	// Store blob if it is new
	blob, err := NewBlobFromDisk(blobID)
	if blob == nil {
		if req.Body == nil || len(req.Body) == 0 {
			// TODO(ortutay): Neither "OK" nor "error" are appropriate status codes
			// in this case. It may be useful to have a third error class, but not
			// sure what to call it.
			return msg.NewRespOk([]byte("Please re-send with data.")), nil
		}
		if len(req.Body) > MAX_BLOB_BYTES {
			resp := msg.NewRespErrorWithBody(msg.CANNOT_COMPLETE_REQUEST,
				[]byte(fmt.Sprintf("Cannot store over %v",
					util.ByteSize(MAX_BLOB_BYTES).String())))
			return resp, nil
		}
		blob, err = NewBlobFromReader(bytes.NewReader(req.Body))
		if err != nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
		err := storeBlob(blob)
		if err != nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
	}
	
	// Append blob-id to container-id
	container := NewContainerFromDisk(req.ID)
	for _, id := range container.BlobIDs {
		if id == blob.ID {
			return msg.NewRespOk([]byte("")), nil
		}
	}
	container.WriteNewBlobID(blob.ID)

	return msg.NewRespOk([]byte(blob.ID.String())), nil
}

// func (ss *StoreService) diff(req *msg.OcReq) (*msg.OcResp, error) {
// 	return nil, nil
// }

func (ss *StoreService) get(req *msg.OcReq) (*msg.OcResp, error) {
	var containerID ContainerID
	var blobID BlobID
	if len(req.Args) == 1 {
		containerID = ocIDToContainerID(req.ID)
		blobID = BlobID(req.Args[0])
	} else if len(req.Args) == 2 {
		if req.Args[0] == "." {
			containerID = ocIDToContainerID(req.ID)
		} else {
			containerID = ContainerID(req.Args[0])
		}
		blobID = BlobID(req.Args[1])
	} else {
		return msg.NewRespError(msg.INVALID_ARGUMENTS), nil
	}

	fmt.Printf("get %v %v\n", containerID, blobID)

	if containerID != ocIDToContainerID(req.ID) {
		resp := msg.NewRespErrorWithBody(msg.INVALID_ARGUMENTS,
			[]byte("Cannot access that container"))
		return resp, nil
	}

	container := NewContainerFromDisk(req.ID)
	if !container.HasBlobID(blobID) {
		resp := msg.NewRespErrorWithBody(msg.INVALID_ARGUMENTS,
			[]byte("Cannot access that blob"))
		return resp, nil
	}

	blob, err := NewBlobFromDisk(blobID)
	if err != nil {
		return msg.NewRespError(msg.SERVER_ERROR), nil
	}
	var buf bytes.Buffer
	for _, block := range blob.Blocks {
		_, err := buf.Write(block.Data)
		if err != nil {
			return msg.NewRespError(msg.SERVER_ERROR), nil
		}
	}
	return msg.NewRespOk(buf.Bytes()), nil
}

func (ss *StoreService) hash(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}
