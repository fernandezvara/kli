package kli

import (
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewSecret(t *testing.T) {
	// Test with value
	s := newSecret("my-secret-value")
	if s == nil {
		t.Fatal("newSecret should not return nil")
	}
	if !s.IsSet() {
		t.Error("Secret should be set")
	}
	defer s.Destroy()

	// Test with empty value
	s2 := newSecret("")
	if s2 == nil {
		t.Fatal("newSecret with empty string should not return nil")
	}
	if s2.IsSet() {
		t.Error("Secret with empty value should not be set")
	}
}

func TestSecretBytes(t *testing.T) {
	s := newSecret("test-secret")
	defer s.Destroy()

	bytes := s.Bytes()
	if string(bytes) != "test-secret" {
		t.Errorf("Secret.Bytes() = %s, expected 'test-secret'", string(bytes))
	}

	// Test nil buffer
	s2 := &Secret{}
	if s2.Bytes() != nil {
		t.Error("Secret.Bytes() should return nil for nil buffer")
	}
}

func TestSecretString(t *testing.T) {
	s := newSecret("test-secret")
	defer s.Destroy()

	str := s.String()
	if str != "test-secret" {
		t.Errorf("Secret.String() = %s, expected 'test-secret'", str)
	}

	// Test nil buffer
	s2 := &Secret{}
	if s2.String() != "" {
		t.Error("Secret.String() should return empty string for nil buffer")
	}
}

func TestSecretDestroy(t *testing.T) {
	s := newSecret("test-secret")

	// Verify it's set before destroy
	if !s.IsSet() {
		t.Error("Secret should be set before destroy")
	}

	s.Destroy()

	// Verify it's not set after destroy
	if s.IsSet() {
		t.Error("Secret should not be set after destroy")
	}
	if s.Size() != 0 {
		t.Error("Secret size should be 0 after destroy")
	}

	// Double destroy should not panic
	s.Destroy()
}

func TestSecretIsSet(t *testing.T) {
	// Set secret
	s := newSecret("value")
	if !s.IsSet() {
		t.Error("Secret with value should be set")
	}
	s.Destroy()

	// Empty secret
	s2 := newSecret("")
	if s2.IsSet() {
		t.Error("Secret with empty value should not be set")
	}

	// Nil buffer
	s3 := &Secret{}
	if s3.IsSet() {
		t.Error("Secret with nil buffer should not be set")
	}
}

func TestSecretSize(t *testing.T) {
	s := newSecret("12345")
	defer s.Destroy()

	if s.Size() != 5 {
		t.Errorf("Secret.Size() = %d, expected 5", s.Size())
	}

	// Nil buffer
	s2 := &Secret{}
	if s2.Size() != 0 {
		t.Error("Secret.Size() should return 0 for nil buffer")
	}
}

func TestSecretStore(t *testing.T) {
	ss := newSecretStore()

	// Store secrets
	ss.Store("key1", "value1")
	ss.Store("key2", "value2")

	// Get existing secrets
	s1 := ss.Get("key1")
	if !s1.IsSet() || s1.String() != "value1" {
		t.Error("SecretStore.Get('key1') failed")
	}

	s2 := ss.Get("key2")
	if !s2.IsSet() || s2.String() != "value2" {
		t.Error("SecretStore.Get('key2') failed")
	}

	// Get non-existent secret
	s3 := ss.Get("nonexistent")
	if s3.IsSet() {
		t.Error("SecretStore.Get('nonexistent') should return unset secret")
	}

	// Destroy all
	ss.DestroyAll()

	// After destroy, getting should return unset secrets
	s4 := ss.Get("key1")
	if s4.IsSet() {
		t.Error("SecretStore.Get after DestroyAll should return unset secret")
	}
}

func TestSecretStoreOverwrite(t *testing.T) {
	ss := newSecretStore()

	ss.Store("key", "value1")
	s1 := ss.Get("key")
	if s1.String() != "value1" {
		t.Error("Initial store failed")
	}

	// Overwrite
	ss.Store("key", "value2")
	s2 := ss.Get("key")
	if s2.String() != "value2" {
		t.Error("Overwrite failed")
	}

	ss.DestroyAll()
}

func TestSecretStoreDestroyAllMultipleTimes(t *testing.T) {
	ss := newSecretStore()

	ss.Store("key1", "value1")
	ss.Store("key2", "value2")

	// Multiple destroys should not panic
	ss.DestroyAll()
	ss.DestroyAll()
	ss.DestroyAll()

	// Should be able to store again after destroy
	ss.Store("key3", "value3")
	s := ss.Get("key3")
	if !s.IsSet() || s.String() != "value3" {
		t.Error("Store after DestroyAll failed")
	}

	ss.DestroyAll()
}

// TestSecretRAII tests automatic cleanup with finalizers
func TestSecretRAII(t *testing.T) {
	// Create a secret
	s := newSecret("test-secret-raii")
	if !s.IsSet() {
		t.Error("Secret should be set initially")
	}

	// Destroy it
	s.Destroy()

	// Verify it's destroyed
	if !s.isDestroyed() {
		t.Error("Secret should be marked as destroyed")
	}

	if s.IsSet() {
		t.Error("Destroyed secret should not be set")
	}

	if s.Size() != 0 {
		t.Error("Destroyed secret should have size 0")
	}

	if s.String() != "" {
		t.Error("Destroyed secret should return empty string")
	}

	if s.Bytes() != nil {
		t.Error("Destroyed secret should return nil bytes")
	}
}

// TestSecretThreadSafety tests concurrent access to secrets
func TestSecretThreadSafety(t *testing.T) {
	ss := newSecretStore()

	// Store multiple secrets
	for i := range 10 {
		ss.Store(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d", i))
	}

	var wg sync.WaitGroup
	errs := make(chan error, 20)

	// Concurrent reads
	for i := range 10 {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			s := ss.Get(fmt.Sprintf("key%d", index))
			if !s.IsSet() || s.String() != fmt.Sprintf("value%d", index) {
				errs <- fmt.Errorf("concurrent read failed for key%d", index)
			}
		}(i)
	}

	// Concurrent writes
	for i := 10; i < 20; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ss.Store(fmt.Sprintf("key%d", index), fmt.Sprintf("value%d", index))
			s := ss.Get(fmt.Sprintf("key%d", index))
			if !s.IsSet() || s.String() != fmt.Sprintf("value%d", index) {
				errs <- fmt.Errorf("concurrent write failed for key%d", index)
			}
		}(i)
	}

	wg.Wait()
	close(errs)

	// Check for errors
	for err := range errs {
		t.Error(err)
	}

	ss.DestroyAll()
}

// TestSecretMemoryLeak tests for memory leaks under pressure
func TestSecretMemoryLeak(t *testing.T) {
	ss := newSecretStore()

	// Create many secrets to test memory pressure
	for i := range 1000 {
		ss.Store(fmt.Sprintf("key%d", i), fmt.Sprintf("value%d-with-longer-content-to-test-memory-management", i))
	}

	// Verify secrets are stored
	if !ss.Has("key999") {
		t.Error("Expected secrets to be stored")
	}

	// Destroy all
	ss.DestroyAll()

	// Verify cleanup
	if len(ss.secrets) != 0 {
		t.Error("SecretStore should have no secrets after destroy")
	}

	// Force garbage collection to trigger finalizers
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Allow finalizers to run
	runtime.GC()

	// Verify store is still destroyed
	if ss.destroyed.Load() != 1 {
		t.Error("SecretStore should still be marked as destroyed")
	}
}

// TestSecretStoreVerification tests cleanup behavior
func TestSecretStoreVerification(t *testing.T) {
	ss := newSecretStore()

	// Initially not destroyed
	if ss.destroyed.Load() != 0 {
		t.Error("New SecretStore should not be destroyed")
	}

	// Add some secrets
	ss.Store("key1", "value1")
	ss.Store("key2", "value2")

	// Test Has method
	if !ss.Has("key1") {
		t.Error("SecretStore should have key1")
	}

	if ss.Has("nonexistent") {
		t.Error("SecretStore should not have nonexistent key")
	}

	// Destroy and verify
	ss.DestroyAll()

	if ss.destroyed.Load() != 1 {
		t.Error("SecretStore should be destroyed")
	}

	if ss.Has("key1") {
		t.Error("Destroyed SecretStore should not have any keys")
	}

	if len(ss.secrets) != 0 {
		t.Error("Destroyed SecretStore should have no keys")
	}
}
