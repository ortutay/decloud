package store

import (
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
	ID ContainerID
	OwnerID msg.OcID
	BlobIDs []BlobID
}

func NewContainerFromDisk(id msg.OcID) *Container {
	d := util.GetOrCreateDB(containerToBlobsDB())
	containerID := ocIDToContainerID(id)
	blobIDsSer, _ := d.Read(containerID.String())
	var blobIDs []BlobID
	if blobIDsSer == nil || len(blobIDsSer) == 0 {
		blobIDs = make([]BlobID, 0)
	} else {
		err := json.Unmarshal(blobIDsSer, &blobIDs)
		util.Ferr(err)
	}
	return &Container{ID: containerID, OwnerID: id, BlobIDs: blobIDs}
}

func (c *Container) WriteNewBlobID(id BlobID) {
	d := util.GetOrCreateDB(containerToBlobsDB())
	c.BlobIDs = append(c.BlobIDs, id)
	blobIDsSer, err := json.Marshal(c.BlobIDs)
	util.Ferr(err)
	err = d.Write(c.ID.String(), blobIDsSer)
	util.Ferr(err)
}

func containerToBlobsDB() string {
 	return util.ServiceDir(SERVICE_NAME) + "/container-to-blobs.db"
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
}

func (ss *StoreService) Handle(req *msg.OcReq) (*msg.OcResp, error) {
	println(fmt.Sprintf("store got request: %v", req))
	if req.Service != SERVICE_NAME {
		panic(fmt.Sprintf("unexpected service %s", req.Service))
	}

	methods := make(map[string]func(*msg.OcReq) (*msg.OcResp, error))
	methods[ALLOC_METHOD] = ss.alloc
	methods[PUT_METHOD] = ss.put

	if method, ok := methods[req.Method]; ok {
		return method(req)
	} else {
		return msg.NewRespError(msg.METHOD_UNSUPPORTED), nil
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
	if len(req.Args) > 2 {
		return msg.NewRespError(msg.INVALID_ARGUMENTS), nil
	}
	var containerID ContainerID
	if len(req.Args) >= 1 && req.Args[0] != "." {
		containerID = ContainerID(req.Args[0])
	} else {
		containerID = ocIDToContainerID(req.ID)
	}
	var blobID BlobID
	if len(req.Args) == 2 && req.Args[1] != "." {
		blobID = BlobID(req.Args[1])
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
	// validate node-id -> container-id
	// validate container-id -> blob-id
	// return blob
	return nil, nil
}

func (ss *StoreService) hash(req *msg.OcReq) (*msg.OcResp, error) {
	return nil, nil
}
