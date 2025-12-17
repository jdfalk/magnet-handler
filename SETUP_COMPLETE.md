# Magnet Handler - macOS Setup Complete

## ✅ Status: WORKING

Your magnet handler is fully functional and ready to use!

### What's Installed

- **Binary**: `/usr/local/bin/magnet-handler` ✓
- **Configuration**: `~/.magnet-handler.conf` ✓
- **App Bundle**: `~/Applications/Magnet Handler.app` ✓
- **Logs**: `~/.cache/magnet-handler/` ✓

### Configuration

Your Deluge server is configured:
- **Host**: 172.16.2.30
- **Port**: 8112
- **Password**: deluge
- **Label**: audiobooks

### Testing

Run the test script to verify it's working:

```bash
/Users/jdfalk/repos/github.com/jdfalk/magnet-handler/test-magnet-handler.sh
```

### Using with Chrome

When you click a magnet link in Chrome:

1. Chrome will ask "How do you want to open this?"
2. Look for "Magnet Handler" in the list
3. Select it and check "Always use this app"
4. The link will be added to your Deluge server

**Note**: If you don't see "Magnet Handler" as an option:
- The app may need to be added to Chrome's allowed handlers
- Try right-clicking the magnet link and selecting "Open with..."
- Or copy the magnet link and run manually:
  ```bash
  /usr/local/bin/magnet-handler "magnet:?xt=urn:btih:..."
  ```

### Logs

Magnet handler logs all activity to: `~/.cache/magnet-handler/`

View the latest:
```bash
tail -50 $(ls -t ~/.cache/magnet-handler/magnet-handler-*.log | head -1)
```

### Manual Testing

To test manually with a real magnet link:

```bash
/usr/local/bin/magnet-handler "magnet:?xt=urn:btih:f8c7f48dce4dc65d83f6d68c58b47797c6afc9ea&dn=Ubuntu-22.04"
```

### Troubleshooting

If magnet links aren't being captured:

1. **Check the app is installed**:
   ```bash
   ls -la ~/Applications/Magnet\ Handler.app/
   ```

2. **Verify the binary works**:
   ```bash
   /usr/local/bin/magnet-handler --help
   ```

3. **Check logs**:
   ```bash
   ls -lh ~/.cache/magnet-handler/
   tail -30 ~/.cache/magnet-handler/magnet-handler-*.log
   ```

4. **Test with a real magnet link**:
   ```bash
   /usr/local/bin/magnet-handler "magnet:?xt=urn:btih:f8c7f48dce4dc65d83f6d68c58b47797c6afc9ea&dn=Ubuntu"
   ```

### How It Works

1. When you click a magnet link in your browser, it opens the Magnet Handler app
2. The app wrapper calls `/usr/local/bin/magnet-handler` with the magnet URI
3. The handler:
   - Validates the magnet link format
   - Authenticates with your Deluge server
   - Adds the torrent with your configured label
   - Logs everything for debugging
   - Tracks added magnets in a local database

### Config File

Edit your settings here: `~/.magnet-handler.conf`

```json
{
  "deluge_host": "172.16.2.30",
  "deluge_port": "8112",
  "deluge_password": "deluge",
  "deluge_label": "audiobooks",
  "json_path": ""
}
```

---

**The handler is ready!** Try clicking a magnet link in Chrome.
