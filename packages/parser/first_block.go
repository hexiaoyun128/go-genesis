// Copyright 2016 The go-daylight Authors
// This file is part of the go-daylight library.
//
// The go-daylight library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-daylight library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-daylight library. If not, see <http://www.gnu.org/licenses/>.

package parser

import (
	"bytes"
	"encoding/hex"
	"errors"
	"io/ioutil"
	"path/filepath"

	"github.com/GenesisKernel/go-genesis/packages/conf"
	"github.com/GenesisKernel/go-genesis/packages/conf/syspar"
	"github.com/GenesisKernel/go-genesis/packages/consts"
	"github.com/GenesisKernel/go-genesis/packages/converter"
	"github.com/GenesisKernel/go-genesis/packages/crypto"
	"github.com/GenesisKernel/go-genesis/packages/model"
	"github.com/GenesisKernel/go-genesis/packages/smart"
	"github.com/GenesisKernel/go-genesis/packages/utils/tx"

	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
)

// FirstBlockParser is parser wrapper
type FirstBlockParser struct {
	*Parser
}

// ErrFirstBlockHostIsEmpty host for first block is not specified
var ErrFirstBlockHostIsEmpty = errors.New("FirstBlockHost is empty")

// Init first block
func (p *FirstBlockParser) Init() error {
	return nil
}

// Validate first block
func (p *FirstBlockParser) Validate() error {
	return nil
}

// Action is fires first block
func (p *FirstBlockParser) Action() error {
	logger := p.GetLogger()
	data := p.TxPtr.(*consts.FirstBlock)
	keyID := crypto.Address(data.PublicKey)
	if err := model.ExecSchemaEcosystem(nil, 1, keyID, ``, keyID); err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("executing ecosystem schema")
		return p.ErrInfo(err)
	}

	key := &model.Key{
		ID:        keyID,
		PublicKey: data.PublicKey,
		Amount:    decimal.NewFromFloat(consts.FIRST_QDLT).String(),
	}
	key.SetTablePrefix(1)

	if err := model.GetDB(p.DbTransaction).Create(key).Error; err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("inserting default page")
		return p.ErrInfo(err)
	}

	page := &model.Page{
		ID:         1,
		Name:       "default_page",
		Menu:       "default_menu",
		Value:      syspar.SysString("default_ecosystem_page"),
		Conditions: `ContractAccess("@1EditPage")`,
	}
	page.SetTablePrefix("1")

	if err := model.GetDB(p.DbTransaction).Create(page).Error; err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("inserting default page")
		return p.ErrInfo(err)
	}

	menu := &model.Menu{
		ID:         1,
		Name:       "default_menu",
		Value:      syspar.SysString(`default_ecosystem_menu`),
		Title:      "default",
		Conditions: `ContractAccess("@1EditMenu")`,
	}

	menu.SetTablePrefix("1")
	if err := model.GetDB(p.DbTransaction).Create(menu).Error; err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("inserting default menu")
		return p.ErrInfo(err)
	}

	if err := smart.LoadContract(p.DbTransaction, `1`); err != nil {
		return p.ErrInfo(err)
	}
	commission := &model.SystemParameter{Name: `commission_wallet`}
	if err := commission.SaveArray([][]string{{"1", converter.Int64ToStr(keyID)}}); err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("saving commission_wallet array")
		return p.ErrInfo(err)
	}

	if err := syspar.SysUpdate(nil); err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("updating syspar")
		return p.ErrInfo(err)
	}

	syspar.SetFirstBlockData(data)
	return nil
}

// Rollback first block
func (p *FirstBlockParser) Rollback() error {
	return nil
}

// Header is returns first block header
func (p FirstBlockParser) Header() *tx.Header {
	return nil
}

// GetKeyIDFromPrivateKey load KeyID fron PrivateKey file
func GetKeyIDFromPrivateKey() (int64, error) {

	key, err := ioutil.ReadFile(filepath.Join(conf.Config.KeysDir, consts.PrivateKeyFilename))
	if err != nil {
		log.WithFields(log.Fields{"type": consts.IOError, "error": err}).Error("reading private key file")
		return 0, err
	}
	key, err = hex.DecodeString(string(key))
	if err != nil {
		log.WithFields(log.Fields{"type": consts.ConversionError, "error": err}).Error("decoding private key from hex")
		return 0, err
	}
	key, err = crypto.PrivateToPublic(key)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.CryptoError, "error": err}).Error("converting private key to public")
		return 0, err
	}

	return crypto.Address(key), nil
}

// GetDataFromFirstBlock returns data of first block
func GetDataFromFirstBlock() (data *consts.FirstBlock, ok bool) {
	block := &model.Block{}
	isFound, err := block.Get(1)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting record of first block")
		return
	}

	if !isFound {
		return
	}

	pb, err := parseBlock(bytes.NewBuffer(block.Data), true)
	if err != nil {
		log.WithFields(log.Fields{"type": consts.ParserError, "error": err}).Error("parsing data of first block")
		return
	}

	if len(pb.Parsers) == 0 {
		log.WithFields(log.Fields{"type": consts.ParserError}).Error("list of parsers is empty")
		return
	}

	p := pb.Parsers[0]
	data, ok = p.TxPtr.(*consts.FirstBlock)
	if !ok {
		log.WithFields(log.Fields{"type": consts.ParserError}).Error("getting data of first block")
		return
	}

	return
}
