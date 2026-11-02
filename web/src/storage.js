// Move a value from a pre-rename key to its new name, once. The old key is
// removed either way so the browser does not keep two copies.
export function migrateKey(storage, oldKey, newKey) {
  try {
    if (storage.getItem(newKey) == null) {
      const old = storage.getItem(oldKey);
      if (old != null) storage.setItem(newKey, old);
    }
    storage.removeItem(oldKey);
  } catch (e) {
    // Storage blocked: nothing to migrate.
  }
}
