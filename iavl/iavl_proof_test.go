package iavl

import (
	"bytes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"testing"
)

func TestIAVLTreeGetWithProof(t *testing.T) {
	var tree *IAVLTree = NewIAVLTree(0, nil)
	require := require.New(t)
	keys := [][]byte{}
	for _, ikey := range []byte{0x11, 0x32, 0x50, 0x72, 0x99} {
		key := []byte{ikey}
		keys = append(keys, key)
		tree.Set(key, []byte(randstr(8)))
	}
	root := tree.Hash()

	key := []byte{0x32}
	val, existProof, notExistProof, err := tree.GetWithProof(key)
	require.NotEmpty(val)
	require.NotNil(existProof)
	err = existProof.Verify(key, val, root)
	require.Nil(err, "%+v", err)
	require.Nil(notExistProof)
	require.Nil(err)

	key = []byte{0x1}
	val, existProof, notExistProof, err = tree.GetWithProof(key)
	require.Empty(val)
	require.Nil(existProof)
	require.NotNil(notExistProof)
	err = notExistProof.Verify(key, root)
	require.Nil(err, "%+v", err)
	require.Nil(err)
}

func TestIAVLTreeKeyExistsProof(t *testing.T) {
	var tree *IAVLTree = NewIAVLTree(0, nil)

	// should get false for proof with nil root
	_, proof, _ := tree.getWithKeyExistsProof([]byte("foo"))
	assert.Nil(t, proof)

	// insert lots of info and store the bytes
	keys := make([][]byte, 200)
	for i := 0; i < 200; i++ {
		key, value := randstr(20), randstr(200)
		tree.Set([]byte(key), []byte(value))
		keys[i] = []byte(key)
	}

	// query random key fails
	_, proof, _ = tree.getWithKeyExistsProof([]byte("foo"))
	assert.Nil(t, proof)

	// query min key fails
	_, proof, _ = tree.getWithKeyExistsProof([]byte{0})
	assert.Nil(t, proof)

	// valid proof for real keys
	root := tree.Hash()
	for _, key := range keys {
		value, proof, _ := tree.getWithKeyExistsProof(key)
		assert.NotEmpty(t, value)
		if assert.NotNil(t, proof) {
			err := proof.Verify(key, value, root)
			assert.Nil(t, err, "%+v", err)
		}
	}
}

func TestIAVLTreeKeyNotExistsProof(t *testing.T) {
	var tree *IAVLTree = NewIAVLTree(0, nil)

	require := require.New(t)

	proof, err := tree.keyNotExistsProof([]byte{0x1})
	require.Nil(proof, "Proof should be nil for empty tree")
	require.NotNil(err)

	keys := [][]byte{}
	for _, ikey := range []byte{0x11, 0x32, 0x50, 0x72, 0x99} {
		key := []byte{ikey}
		keys = append(keys, key)
		tree.Set(key, []byte(randstr(8)))
	}
	root := tree.Hash()

	// Get min and max keys.
	min, _ := tree.GetByIndex(0)
	max, _ := tree.GetByIndex(tree.Size() - 1)

	// Go through a range of keys and test the result of creating non-existence
	// proofs for them.

	for i := min[0] - 1; i < max[0]+1; i++ {
		key := []byte{i}
		exists := false

		for _, k := range keys {
			if bytes.Compare(key, k) == 0 {
				exists = true
				break
			}
		}

		if exists {
			proof, err = tree.keyNotExistsProof(key)
			require.Nil(proof, "Proof should be nil for existing key")
			require.NotNil(err, "Got verification error for 0x%x: %+v", key, err)
		} else {
			proof, err = tree.keyNotExistsProof(key)
			require.NotNil(proof, "Proof should not be nil for non-existing key")
			require.Nil(err, "%+v", err)

			err = proof.Verify(key, root)
			require.Nil(err, "Got verification error for 0x%x: %+v", key, err)

			if bytes.Compare(key, min) < 0 {
				require.Nil(proof.LeftPath)
				require.NotNil(proof.RightPath)
			} else if bytes.Compare(key, max) > 0 {
				require.Nil(proof.RightPath)
				require.NotNil(proof.LeftPath)
			} else {
				require.NotNil(proof.LeftPath)
				require.NotNil(proof.RightPath)
			}
		}
	}
}

func TestKeyNotExistsProofVerify(t *testing.T) {
	var tree *IAVLTree = NewIAVLTree(0, nil)
	require := require.New(t)
	keys := [][]byte{}
	for _, ikey := range []byte{0x11, 0x32, 0x50, 0x72, 0x99} {
		key := []byte{ikey}
		keys = append(keys, key)
		tree.Set(key, []byte(randstr(8)))
	}

	// Create a bogus non-existence proof and check that it does not verify.

	lkey := keys[0]
	lval, lproof, _ := tree.getWithKeyExistsProof(lkey)
	require.NotNil(lproof)

	rkey := keys[2]
	rval, rproof, _ := tree.getWithKeyExistsProof(rkey)
	require.NotNil(rproof)

	missing := []byte{0x40}

	proof := &KeyNotExistsProof{
		RootHash: lproof.RootHash,

		LeftPath: &lproof.PathToKey,
		LeftNode: IAVLProofLeafNode{KeyBytes: lkey, ValueBytes: lval},

		RightPath: &rproof.PathToKey,
		RightNode: IAVLProofLeafNode{KeyBytes: rkey, ValueBytes: rval},
	}
	err := proof.Verify(missing, tree.Hash())
	require.NotNil(err, "Proof should not verify")

	proof, err = tree.keyNotExistsProof(missing)
	require.Nil(err)
	require.NotNil(proof)

	err = proof.Verify(missing, tree.Hash())
	require.Nil(err)

	err = proof.Verify([]byte{0x45}, tree.Hash())
	require.Nil(err)

	err = proof.Verify([]byte{0x25}, tree.Hash())
	require.NotNil(err)
}