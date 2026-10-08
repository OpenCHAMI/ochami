// SPDX-FileCopyrightText: © 2026 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package config

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	kyaml "github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

// fileParser is the koanf YAML parser used to read and write config files.
var fileParser = kyaml.Parser()

// fileConf is the koanf configuration used by File. Unlike koanfConf (used by
// the loaders), StrictMerge is disabled: editing must tolerate wholesale
// key-type replacement (e.g. swapping the "clusters" list wholesale), which a
// strict merge rejects.
var fileConf = koanf.Conf{Delim: ".", StrictMerge: false}

// File represents a single on-disk configuration file opened for reading and
// editing. Unlike the Load family, which merges multiple sources and applies
// defaults for display, File operates on exactly one file's contents as
// written, with no defaults applied, so that editing never bakes resolved
// default values into the file.
//
// File is the public equivalent of what the "ochami config" and "ochami
// config cluster" commands do; use it to build, inspect, and mutate a
// configuration file programmatically.
//
// Mutations are transactional: every mutating method runs as (or inside) an
// Update, which stages changes on a copy of the in-memory state, persists the
// copy durably, and only then adopts it as f's state. A mutation that rejects
// its input or fails to write therefore leaves both the file on disk and f's
// in-memory state exactly as they were. The one exception is a failure to
// sync the file's directory after the new contents replaced the file: f then
// adopts the new contents, so its state still matches the file, and the
// error is still returned.
type File struct {
	path string
	ko   *koanf.Koanf

	// staging holds the working copy of the Update call in progress, which
	// mutations change and reads observe. It is nil outside Update.
	staging *koanf.Koanf

	// writeOps overrides the durable-write primitives save uses; nil means
	// defaultWriteOps. Tests set it to inject write failures.
	writeOps *fileWriteOps
}

// OpenFile opens the config file at path for reading and editing. The file
// must already exist; use CreateFile to start a new one.
func OpenFile(path string) (*File, error) {
	if path == "" {
		return nil, fmt.Errorf("no configuration file path passed")
	}
	ko := koanf.NewWithConf(fileConf)
	if err := ko.Load(file.Provider(path), fileParser); err != nil {
		return nil, fmt.Errorf("failed to load config file %s: %w", path, err)
	}
	return &File{path: path, ko: ko}, nil
}

// CreateFile creates (or truncates) an empty config file at path and opens it
// for editing.
func CreateFile(path string) (*File, error) {
	if path == "" {
		return nil, fmt.Errorf("no configuration file path passed")
	}
	f := &File{path: path, ko: koanf.NewWithConf(fileConf)}
	if err := f.save(f.ko); err != nil {
		return nil, err
	}
	return f, nil
}

// Path returns the path f was opened from.
func (f *File) Path() string { return f.path }

// Get returns the raw value at key as written in the file, with no defaults
// applied. An empty key returns the same result as Raw. Inside Update it
// reflects the changes made so far.
func (f *File) Get(key string) any {
	if key == "" {
		return f.current().Raw()
	}
	return f.current().Get(key)
}

// Raw returns the entire file's contents as a nested map, with no defaults
// applied. Inside Update it reflects the changes made so far.
func (f *File) Raw() map[string]any { return f.current().Raw() }

// current returns the state that reads observe: the working copy while an
// Update call is in progress, so fn sees its own earlier changes, and the
// committed state otherwise.
func (f *File) current() *koanf.Koanf {
	if f.staging != nil {
		return f.staging
	}
	return f.ko
}

// save marshals ko to YAML and writes it to f.path (or, if f.path is a
// symbolic link, to the file it links to), preserving the file's existing
// mode if it exists. It does not mutate f; Update promotes ko into f.ko only
// once save has succeeded.
func (f *File) save(ko *koanf.Koanf) error {
	b, err := ko.Marshal(fileParser)
	if err != nil {
		return fmt.Errorf("failed to marshal config for writing: %w", err)
	}
	path, err := resolveSymlinks(f.path)
	if err != nil {
		return fmt.Errorf("failed to resolve config file path %s: %w", f.path, err)
	}
	ops := defaultWriteOps
	if f.writeOps != nil {
		ops = *f.writeOps
	}
	return writeFile(path, b, ops)
}

// maxSymlinks bounds how many symbolic links resolveSymlinks follows, so a
// link cycle fails instead of looping forever.
const maxSymlinks = 40

// resolveSymlinks returns the file that path ultimately names, following
// symbolic links (including a final link whose target doesn't exist yet).
// writeFile replaces its destination by renaming over it, which would replace
// a symlinked config file with a regular file; writing to the resolved path
// updates the link's target and leaves the link in place.
func resolveSymlinks(path string) (string, error) {
	for range maxSymlinks {
		finfo, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
		if finfo.Mode()&fs.ModeSymlink == 0 {
			return path, nil
		}
		target, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		path = target
	}
	return "", fmt.Errorf("%s: too many levels of symbolic links", path)
}

// temporaryFile is the subset of *os.File that writeFile needs, narrow enough
// to fake in tests without touching a real filesystem.
type temporaryFile interface {
	io.Writer
	Chmod(os.FileMode) error
	Sync() error
	Close() error
	Name() string
}

// fileWriteOps are the durable-write primitives writeFile uses, injected so
// each failure point (create, chmod, write, sync, close, rename, directory
// sync) can be exercised in tests without depending on real filesystem
// failure conditions.
type fileWriteOps struct {
	stat       func(string) (os.FileInfo, error)
	createTemp func(string, string) (temporaryFile, error)
	rename     func(string, string) error
	remove     func(string) error
	syncDir    func(string) error
}

// createTemporaryFile is the production adapter for creating a temporary
// file.
func createTemporaryFile(dir, pattern string) (temporaryFile, error) {
	return os.CreateTemp(dir, pattern)
}

// defaultWriteOps are the durable-write primitives File uses in production.
var defaultWriteOps = fileWriteOps{
	stat:       os.Stat,
	createTemp: createTemporaryFile,
	rename:     os.Rename,
	remove:     os.Remove,
	syncDir:    syncParentDirectory,
}

// dirSyncError reports that writeFile replaced the file but couldn't sync
// its parent directory: the file holds the new contents, but the rename may
// not survive a crash.
type dirSyncError struct {
	path string
	err  error
}

func (e *dirSyncError) Error() string {
	return fmt.Sprintf("failed to sync parent directory for config file %s: %v", e.path, e.err)
}

func (e *dirSyncError) Unwrap() error { return e.err }

// writeFile durably writes data to path: it writes beside the destination
// through a synced, closed temporary file, renames it into place, and syncs
// the parent directory, so a crash or power loss during the write can never
// leave path corrupt, half-written, or (after the rename) pointing at a
// directory entry that didn't survive the crash. The file's existing mode is
// preserved if it exists.
func writeFile(path string, data []byte, ops fileWriteOps) error {
	var fmode os.FileMode = 0o644
	if finfo, err := ops.stat(path); err == nil {
		fmode = finfo.Mode().Perm()
	}

	tmp, err := ops.createTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return fmt.Errorf("failed to create temporary config file for %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	defer ops.remove(tmpPath) //nolint:errcheck // best-effort cleanup after rename or failure
	if err := writeTemporaryConfig(tmp, fmode, data); err != nil {
		return fmt.Errorf("failed to write temporary config file for %s: %w", path, err)
	}

	if err := ops.rename(tmpPath, path); err != nil {
		return fmt.Errorf("failed to replace config file %s: %w", path, err)
	}
	if err := ops.syncDir(filepath.Dir(path)); err != nil {
		return &dirSyncError{path: path, err: err}
	}

	return nil
}

// writeTemporaryConfig writes and durably closes a previously created
// temporary file. On failures before Close, it makes a best-effort Close so
// the caller can remove the file.
func writeTemporaryConfig(tmp temporaryFile, mode os.FileMode, data []byte) (retErr error) {
	closeAttempted := false
	defer func() {
		if retErr != nil && !closeAttempted {
			_ = tmp.Close()
		}
	}()
	if err := tmp.Chmod(mode); err != nil {
		return fmt.Errorf("set permissions: %w", err)
	}
	n, err := tmp.Write(data)
	if err != nil {
		return fmt.Errorf("write data: %w", err)
	}
	if n != len(data) {
		return fmt.Errorf("write data: %w", io.ErrShortWrite)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync data: %w", err)
	}
	closeAttempted = true
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close file: %w", err)
	}
	return nil
}

// Update runs fn, which may call any of File's mutating methods, and saves
// the file to disk exactly once after fn returns successfully, instead of
// once per call within fn. Use it to combine multiple changes (e.g.
// SetClusterKey followed by SetDefaultCluster) into a single write.
// Each mutating method runs through Update itself, so a call made outside one
// is a batch of one, and a nested call joins the outer batch.
//
// fn's mutations accumulate on a working copy, which Get, Raw, Cluster, and
// Clusters read while fn runs, so fn observes its own earlier changes. If fn
// returns an error, nothing is written to disk and f's in-memory state is
// left exactly as it was before Update was called. A failed save likewise
// discards the copy, so f's state never reflects a change that did not reach
// disk. If the new contents replaced the file but its directory couldn't be
// synced, f adopts the copy to stay in step with the file, and Update still
// returns the error.
func (f *File) Update(fn func(f *File) error) error {
	if f.staging != nil {
		// Already inside an outer Update call; let it save.
		return fn(f)
	}
	f.staging = f.ko.Copy()
	defer func() { f.staging = nil }()
	if err := fn(f); err != nil {
		return err
	}
	if err := f.save(f.staging); err != nil {
		var syncErr *dirSyncError
		if errors.As(err, &syncErr) {
			// The new contents already replaced the file; only the rename's
			// durability is in doubt. Keep f in step with the file.
			f.ko = f.staging
		}
		return fmt.Errorf("failed to write modified config to %s: %w", f.path, err)
	}
	f.ko = f.staging
	return nil
}

// clustersKey is the top-level key File's dedicated cluster methods (
// AddCluster, SetClusterKey, etc.) own. SetKey and UnsetKey refuse to touch
// it directly so a caller can't bypass those methods' validation (name
// uniqueness, delimiter checks) by writing to "clusters" as a generic key.
const clustersKey = "clusters"

// isClustersKey reports whether key is "clusters" or nested under it.
func isClustersKey(ko *koanf.Koanf, key string) bool {
	return key == clustersKey || strings.HasPrefix(key, clustersKey+ko.Delim())
}

// SetKey sets key to value and saves the file. key must not be "clusters" or
// nested under it; use AddCluster/SetClusterKey/RenameCluster/etc for cluster
// configuration.
func (f *File) SetKey(key string, value any) error {
	return f.Update(func(*File) error {
		ko := f.staging
		if isClustersKey(ko, key) {
			return ErrInvalidConfigVal{
				Key:      key,
				Value:    fmt.Sprint(value),
				Expected: "a non-cluster key; use AddCluster/SetClusterKey/RenameCluster/etc for cluster configuration",
			}
		}
		if err := ko.Set(key, value); err != nil {
			return fmt.Errorf("failed to set key %s to value %v: %w", key, value, err)
		}
		return nil
	})
}

// UnsetKey removes key and saves the file. It returns ErrUnknownKey if key is
// not set. key must not be "clusters" or nested under it; use
// UnsetClusterKey/DeleteCluster for cluster configuration.
func (f *File) UnsetKey(key string) error {
	return f.Update(func(*File) error {
		ko := f.staging
		if isClustersKey(ko, key) {
			return ErrInvalidConfigVal{
				Key:      key,
				Value:    "",
				Expected: "a non-cluster key; use UnsetClusterKey/DeleteCluster for cluster configuration",
			}
		}
		if !ko.Exists(key) {
			return ErrUnknownKey{Key: key}
		}
		ko.Delete(key)
		return nil
	})
}

// checkClusterName rejects an empty cluster name or one containing koanf's
// key delimiter, which would otherwise be misinterpreted as a nested path.
func checkClusterName(ko *koanf.Koanf, name string) error {
	if name == "" {
		return ErrInvalidConfigVal{
			Key:      "name",
			Value:    name,
			Expected: "a non-empty cluster name",
		}
	}
	delim := ko.Delim()
	if strings.Contains(name, delim) {
		return ErrInvalidConfigVal{
			Key:      "name",
			Value:    name,
			Expected: fmt.Sprintf("cluster name without delimiter %q", delim),
		}
	}
	return nil
}

// clusterIndex returns the index of the cluster named name within clusters,
// or -1 if no such cluster exists.
func clusterIndex(clusters []map[string]any, name string) int {
	for i, cl := range clusters {
		if cl["name"] == name {
			return i
		}
	}
	return -1
}

// rawClusters returns the staged "clusters" list as raw maps, in file order.
// It reads the working copy of the Update in progress, so mutations
// batched under Update observe each other's changes.
func (f *File) rawClusters() ([]map[string]any, error) {
	var clusters []map[string]any
	if err := f.staging.Unmarshal("clusters", &clusters); err != nil {
		return nil, fmt.Errorf("unable to unmarshal clusters: %w", err)
	}
	return clusters, nil
}

// setClusters replaces the staged "clusters" list.
func (f *File) setClusters(clusters []map[string]any) error {
	if err := f.staging.Set("clusters", clusters); err != nil {
		return fmt.Errorf("unable to set clusters: %w", err)
	}
	return nil
}

// Cluster returns the named cluster's configuration, with no defaults
// applied. It returns ErrUnknownCluster if no such cluster exists. Inside
// Update it reflects the changes made so far.
func (f *File) Cluster(name string) (Cluster, error) {
	clusters, err := f.Clusters()
	if err != nil {
		return Cluster{}, err
	}
	for _, cl := range clusters {
		if cl.Name == name {
			return cl, nil
		}
	}
	return Cluster{}, ErrUnknownCluster{ClusterName: name}
}

// Clusters returns every cluster in the file, with no defaults applied, in
// file order. Inside Update it reflects the changes made so far.
func (f *File) Clusters() ([]Cluster, error) {
	var clusters []Cluster
	if err := f.current().Unmarshal("clusters", &clusters); err != nil {
		return nil, fmt.Errorf("unable to unmarshal clusters: %w", err)
	}
	return clusters, nil
}

// AddCluster adds a new cluster with the given name and configuration and
// saves the file. It returns ErrClusterExists if a cluster with that name
// already exists.
//
// Because this writes cfg's fields in full, any field left at its zero value
// is written explicitly (for example enable-auth: false, rather than being
// left unset so the built-in default of true applies on load). Callers who
// want default-inheriting creation should build the cluster up with
// SetClusterKey instead, which only ever writes the keys it's given.
func (f *File) AddCluster(name string, cfg ClusterConfig) error {
	return f.Update(func(*File) error {
		if err := checkClusterName(f.staging, name); err != nil {
			return err
		}
		clusters, err := f.rawClusters()
		if err != nil {
			return err
		}
		if clusterIndex(clusters, name) != -1 {
			return ErrClusterExists{ClusterName: name}
		}

		kc := koanf.NewWithConf(fileConf)
		if err := kc.Load(structs.Provider(cfg, "koanf"), nil); err != nil {
			return fmt.Errorf("unable to load cluster config: %w", err)
		}
		clusters = append(clusters, map[string]any{
			"name":    name,
			"cluster": kc.Raw(),
		})

		if err := f.setClusters(clusters); err != nil {
			return err
		}
		return nil
	})
}

// SetClusterKey sets key within the named cluster's configuration to value,
// creating the cluster if it doesn't already exist, and saves the file. key
// must not be "name"; use RenameCluster to rename a cluster.
func (f *File) SetClusterKey(name, key string, value any) error {
	return f.Update(func(*File) error {
		if key == "name" {
			return ErrInvalidConfigVal{
				Key:      key,
				Value:    key,
				Expected: "a key other than \"name\"; use RenameCluster to rename a cluster",
			}
		}
		if err := checkClusterName(f.staging, name); err != nil {
			return err
		}
		clusters, err := f.rawClusters()
		if err != nil {
			return err
		}

		cidx := clusterIndex(clusters, name)
		if cidx == -1 {
			cidx = len(clusters)
			clusters = append(clusters, map[string]any{"name": name})
		}

		kc := koanf.NewWithConf(fileConf)
		if err := kc.Load(confmap.Provider(clusters[cidx], ""), nil); err != nil {
			return fmt.Errorf("unable to load cluster '%s': %w", name, err)
		}
		if err := kc.Set(key, value); err != nil {
			return fmt.Errorf("unable to set %s in cluster '%s': %w", key, name, err)
		}
		clusters[cidx] = kc.Raw()

		if err := f.setClusters(clusters); err != nil {
			return err
		}
		return nil
	})
}

// UnsetClusterKey removes key from the named cluster's configuration and
// saves the file. key must not be "name". It returns ErrUnknownCluster if the
// cluster doesn't exist, or ErrUnknownKey if the key within it doesn't exist.
func (f *File) UnsetClusterKey(name, key string) error {
	return f.Update(func(*File) error {
		if key == "name" {
			return ErrInvalidConfigVal{
				Key:      key,
				Value:    key,
				Expected: "a key other than \"name\"; a cluster's name cannot be unset",
			}
		}
		if err := checkClusterName(f.staging, name); err != nil {
			return err
		}
		clusters, err := f.rawClusters()
		if err != nil {
			return err
		}

		cidx := clusterIndex(clusters, name)
		if cidx == -1 {
			return ErrUnknownCluster{ClusterName: name}
		}

		kc := koanf.NewWithConf(fileConf)
		if err := kc.Load(confmap.Provider(clusters[cidx], ""), nil); err != nil {
			return fmt.Errorf("unable to load cluster '%s': %w", name, err)
		}
		if !kc.Exists(key) {
			return ErrUnknownKey{Key: key}
		}
		kc.Delete(key)
		clusters[cidx] = kc.Raw()

		if err := f.setClusters(clusters); err != nil {
			return err
		}
		return nil
	})
}

// RenameCluster renames the cluster named oldName to newName and saves the
// file. If the renamed cluster was the default cluster, default-cluster is
// updated to newName. It returns ErrUnknownCluster if oldName doesn't exist,
// or ErrClusterExists if newName is already taken.
func (f *File) RenameCluster(oldName, newName string) error {
	return f.Update(func(*File) error {
		if err := checkClusterName(f.staging, newName); err != nil {
			return err
		}
		clusters, err := f.rawClusters()
		if err != nil {
			return err
		}

		if clusterIndex(clusters, newName) != -1 {
			return ErrClusterExists{ClusterName: newName}
		}
		cidx := clusterIndex(clusters, oldName)
		if cidx == -1 {
			return ErrUnknownCluster{ClusterName: oldName}
		}

		wasDefault := f.staging.String("default-cluster") == oldName
		clusters[cidx]["name"] = newName
		if err := f.setClusters(clusters); err != nil {
			return err
		}

		if wasDefault {
			if err := f.staging.Set("default-cluster", newName); err != nil {
				return fmt.Errorf("failed to update default-cluster: %w", err)
			}
		}

		return nil
	})
}

// SetDefaultCluster sets default-cluster to name and saves the file. It
// returns ErrUnknownCluster if no such cluster exists.
func (f *File) SetDefaultCluster(name string) error {
	return f.Update(func(*File) error {
		clusters, err := f.rawClusters()
		if err != nil {
			return err
		}
		if clusterIndex(clusters, name) == -1 {
			return ErrUnknownCluster{ClusterName: name}
		}
		if err := f.staging.Set("default-cluster", name); err != nil {
			return fmt.Errorf("failed to set default-cluster: %w", err)
		}
		return nil
	})
}

// DeleteCluster removes the named cluster entirely and saves the file,
// clearing default-cluster if it pointed at the deleted cluster. It returns
// ErrUnknownCluster if the cluster does not exist.
func (f *File) DeleteCluster(name string) error {
	return f.Update(func(*File) error {
		clusters, err := f.rawClusters()
		if err != nil {
			return err
		}

		cidx := clusterIndex(clusters, name)
		if cidx == -1 {
			return ErrUnknownCluster{ClusterName: name}
		}
		newClusters := make([]map[string]any, 0, len(clusters)-1)
		newClusters = append(newClusters, clusters[:cidx]...)
		newClusters = append(newClusters, clusters[cidx+1:]...)

		if err := f.setClusters(newClusters); err != nil {
			return err
		}

		if f.staging.String("default-cluster") == name {
			f.staging.Delete("default-cluster")
		}

		return nil
	})
}
