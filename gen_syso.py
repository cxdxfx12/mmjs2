"""Generate Windows .syso resource file from ICO for Go embedding."""
import struct
import os

ICO_PATH = "monkey.ico"
OUT_PATH = "rsrc_windows_amd64.syso"

# Machine types
IMAGE_FILE_MACHINE_AMD64 = 0x8664

# Section characteristics
IMAGE_SCN_CNT_INITIALIZED_DATA = 0x00000040
IMAGE_SCN_MEM_READ = 0x40000000
IMAGE_SCN_LNK_INFO = 0x00000200
IMAGE_SCN_LNK_REMOVE = 0x00000800

# Resource types
RT_ICON = 3
RT_GROUP_ICON = 14


def pad_to_4(n):
    return (n + 3) & ~3


def build_resource_tree(icon_data):
    """Build the Windows resource directory tree."""
    
    # Parse ICO file to get individual icon entries
    # ICO header: reserved(2) + type(2) + count(2)
    if len(icon_data) < 6:
        raise ValueError("ICO file too small")
    
    reserved, img_type, count = struct.unpack_from("<HHH", icon_data, 0)
    if img_type != 1:  # ICO type
        raise ValueError("Not a valid ICO file")
    
    # Parse icon directory entries
    icons = []
    offset = 6
    for i in range(count):
        w, h, colors, reserved2, planes, bpp, size, data_offset = struct.unpack_from(
            "<BBBBHHII", icon_data, offset
        )
        # width: 0 means 256
        if w == 0:
            w = 256
        if h == 0:
            h = 256
        icons.append({
            'width': w,
            'height': h,
            'colors': colors,
            'planes': planes,
            'bpp': bpp,
            'size': size,
            'offset': data_offset,
            'data': icon_data[data_offset:data_offset + size],
            'id': i + 1  # icon IDs start at 1
        })
        offset += 16
    
    data_entries = []  # list of (rva, data) tuples for .rsrc$02
    data_rva = 0  # relative virtual address within .rsrc$02
    
    def add_data(data_block):
        nonlocal data_rva
        rva = data_rva
        data_rva += len(data_block)
        data_entries.append((rva, data_block))
        return rva
    
    # ---------- Build resource tree ----------
    # We need:
    # Root -> Type (RT_ICON=3) -> ID (1..n) -> Language (0x0804) -> Data
    # Root -> Type (RT_GROUP_ICON=14) -> ID (1) -> Language (0x0804) -> Data
    
    def make_directory(entries):
        """entries: list of (id_or_name, is_name, subdirectory_or_data_rva)"""
        buf = bytearray()
        # IMAGE_RESOURCE_DIRECTORY
        num_named = sum(1 for e in entries if e[1])  # name entries
        num_id = sum(1 for e in entries if not e[1])  # ID entries
        buf += struct.pack("<IIHHHH", 0, 0, 0, 0, num_named, num_id)
        
        entry_buf = bytearray()
        string_buf = bytearray()
        next_string_offset = 0
        
        for name_or_id, is_name, sub in entries:
            if is_name:
                # Name entry: high bit set, offset to name string
                name_str = name_or_id
                name_bytes = name_str.encode('utf-16-le')
                # Store string
                string_off = len(buf) + len(entry_buf) + 8 * len(entries) + next_string_offset
                name_size = 2 + len(name_bytes)  # length prefix + string
                string_entry = struct.pack("<H", len(name_str)) + name_bytes
                string_buf += string_entry
                # Name RVA: high bit set
                entry_buf += struct.pack("<I", string_off | 0x80000000)
                next_string_offset += name_size
            else:
                entry_buf += struct.pack("<I", name_or_id)
            
            if isinstance(sub, tuple):
                # data entry (OffsetToData)
                data_rva_val, data_size = sub
                entry_buf += struct.pack("<I", data_rva_val & 0x7FFFFFFF)
            else:
                # subdirectory (OffsetToDirectory: high bit set)
                entry_buf += struct.pack("<I", sub | 0x80000000)
        
        # Sort entries: named first, then by ID
        # For simplicity, just append in order
        
        full = bytearray()
        full += buf
        full += entry_buf
        full += string_buf
        return bytes(full)
    
    def make_data_entry(rva, size):
        return struct.pack("<IIII", rva, size, 1200, 0)  # CodePage 1200 = Unicode
    
    # Build from bottom up
    
    # For each icon: LANGUAGE -> DATA
    language_dirs = []
    lang_entry_offsets = []
    
    for icon in icons:
        # Store raw icon data
        icon_rva = add_data(icon['data'])
        data_entry = make_data_entry(icon_rva, icon['size'])
        # Store data entry
        de_rva = add_data(data_entry)
        # Language directory: 1 ID entry (0x0804 = Chinese Simplified)
        lang_dir = make_directory([(0x0804, False, (de_rva, icon['size']))])
        ld_rva = add_data(lang_dir)
        language_dirs.append(ld_rva)
        lang_entry_offsets.append(ld_rva)
    
    # ICON ID directories: one for each icon
    icon_id_dirs = []
    for i, ld_rva in enumerate(language_dirs):
        id_dir = make_directory([(icons[i]['id'], False, ld_rva)])
        id_rva = add_data(id_dir)
        icon_id_dirs.append(id_rva)
    
    # RT_ICON type directory
    icon_type_entries = [(icons[i]['id'], False, icon_id_dirs[i]) for i in range(len(icons))]
    icon_type_dir = make_directory(icon_type_entries)
    icon_type_rva = add_data(icon_type_dir)
    
    # RT_GROUP_ICON: Build the group icon data
    # Group icon header: same as ICO header
    group_icon_data = bytearray()
    group_icon_data += struct.pack("<HHH", 0, 1, len(icons))  # reserved, type=1, count
    
    rva_base = 0  # Will be calculated, but in our case it's just data offset
    data_offset_in_group = 6 + 14 * len(icons)
    for icon in icons:
        group_icon_data += struct.pack("<BBBBHHIH",
            icon['width'] if icon['width'] != 256 else 0,
            icon['height'] if icon['height'] != 256 else 0,
            icon['colors'],
            0,  # reserved
            icon['planes'],
            icon['bpp'],
            icon['size'],
            icon['id']  # This is the RT_ICON resource ID
        )
    
    group_icon_size = len(group_icon_data)
    group_rva = add_data(bytes(group_icon_data))
    
    # Group icon data entry
    group_de = make_data_entry(group_rva, group_icon_size)
    group_de_rva = add_data(group_de)
    
    # Group icon language dir
    group_lang_dir = make_directory([(0x0804, False, (group_de_rva, group_icon_size))])
    group_lang_rva = add_data(group_lang_dir)
    
    # Group icon ID dir (ID=1)
    group_id_dir = make_directory([(1, False, group_lang_rva)])
    group_id_rva = add_data(group_id_dir)
    
    # RT_GROUP_ICON type dir
    group_type_dir = make_directory([(RT_GROUP_ICON, False, group_id_rva)])
    group_type_rva = add_data(group_type_dir)
    
    # Root directory: 2 ID entries
    root_dir = make_directory([
        (RT_ICON, False, icon_type_rva),
        (RT_GROUP_ICON, False, group_type_rva),
    ])
    root_rva = add_data(root_dir)
    
    # .rsrc$01 = resource directory tree (all directories + data entries)
    # .rsrc$02 = raw data (icon data + group icon data)
    
    dir_data = bytearray()
    raw_data = bytearray()
    
    # Separate directories/strings from raw data
    # Everything before raw icon data goes to $01
    # Actually, the convention is:
    # $01 = resource directory tree (directories, entries, strings)
    # $02 = raw resource data (icon bytes, group icon header)
    
    # We'll put everything into one block and separate based on content type
    # Simpler approach: all data_entries are in .rsrc$01, icon data in .rsrc$02
    # But actually both are combined... Let me just interleave them properly.
    
    # Actually let me simplify: put EVERYTHING in $01 side-by-side with proper alignment
    # and put icon data in $02 with $01 referencing $02 by offset.
    
    # Let me restructure:
    all_resource_tree = bytearray()
    all_raw_data = bytearray()
    
    icon_data_start = 0
    for rva, block in data_entries:
        # Check if this is actual icon data or resource tree data
        # Directory entries and data entries go to $01
        # Actual pixel data goes to $02
        is_raw = len(block) > 200  # Heuristic: icon pixel data is big
        # Better approach: determine by content
        # Actually let me just put everything in one section and have $02 be the ICON data
        
        if is_raw:
            all_raw_data += block
        else:
            all_resource_tree += block
    
    # Actually, this heuristic is bad. Let me be more precise.
    # The standard approach is: 
    # .rsrc$01 contains the resource directory entries + data entry descriptors
    # .rsrc$02 contains the actual resource data
    
    # Let me rebuild more carefully:
    # All IMAGE_RESOURCE_DIRECTORY structures go to $01
    # All IMAGE_RESOURCE_DATA_ENTRY structures go to $01  
    # All actual resource data (icon pixels, group icon) go to $02
    
    # OK this is getting complex. Let me use the simpler approach:
    # Put ALL directory tree structures + data entries in $01
    # Put ALL icon/cursor raw bytes in $02
    # This works because the linker just appends both sections.
    
    # But the RVAs in data entries need to point within $02...
    # Actually no, in COFF resource objects, the RVA is relative to the start of the 
    # combined resource sections. Both sections are concatenated: $01 first, then $02.
    # So a RVA in $01 that points to the beginning of $02 = sizeof($01)
    
    # OK I think the cleanest approach is to build the whole thing as one blob,
    # then split it into two sections.
    
    full_blob = bytearray()
    for rva, block in data_entries:
        # Pad to alignment
        while len(full_blob) % 4 != 0:
            full_blob.append(0)
        data_entries_list = [e for e in data_entries]
        idx = data_entries_list.index((rva, block))
        # Store the actual offset
        actual_offset = len(full_blob)
        full_blob += block
        # Update RVA in data_entries
        data_entries_list[idx] = (actual_offset, block)  # Doesn't change the tuple though
    
    # Hmm, this is getting tangled. Let me just write a correct implementation
    # using a two-pass approach.
    
    # Pass 1: Build entire resource tree in memory
    # Pass 2: Split into $01 (metadata) and $02 (data) sections
    
    return build_final_syso_2pass(icons, icon_data, count)


def build_final_syso_2pass(icons, icon_data, count):
    """Simplified build: put everything in one big blob with proper structure."""
    
    # Build resource tree in memory
    tree = bytearray()
    data = bytearray()
    
    # We'll build a list of "objects" with types: DIR, DATA_ENTRY, or RAW
    objects = []
    
    def add_obj(obj_type, obj_data, name=""):
        objects.append((obj_type, obj_data, name))
        return len(objects) - 1
    
    results = {}  # name -> index
    
    # ---- RAW ICON DATA ----
    icon_raw_idxs = []
    for icon in icons:
        idx = add_obj("RAW", icon['data'], f"icon_{icon['id']}")
        icon_raw_idxs.append(idx)
    
    # ---- DATA ENTRIES for each icon ----
    icon_de_idxs = []
    for i, icon in enumerate(icons):
        de = struct.pack("<IIII", 0, icon['size'], 1200, 0)  # RVA will be patched
        idx = add_obj("DATA_ENTRY", de, f"de_icon_{icon['id']}")
        icon_de_idxs.append((idx, icon_raw_idxs[i], icon['size']))
    
    # ---- LANGUAGE DIRS for each icon ----
    for i in range(len(icons)):
        lang_dir = make_simple_dir([(0x0804, False, 0)])  # sub will be patched
        idx = add_obj("DIR", lang_dir, f"lang_icon_{i+1}")
        results[f"lang_icon_{i+1}"] = (idx, icon_de_idxs[i][0])
    
    # ---- ID DIRS for each icon ----
    for i in range(len(icons)):
        id_dir = make_simple_dir([(icons[i]['id'], False, 0)])
        idx = add_obj("DIR", id_dir, f"id_icon_{i+1}")
        results[f"id_icon_{i+1}"] = (idx, results[f"lang_icon_{i+1}"][0])
    
    # ---- RT_ICON TYPE DIR ----
    icon_type_entries = []
    for i in range(len(icons)):
        icon_type_entries.append((icons[i]['id'], False, 0))
    icon_type_dir = make_simple_dir(icon_type_entries)
    idx = add_obj("DIR", icon_type_dir, "type_icon")
    results["type_icon"] = (idx, [])
    for i in range(len(icons)):
        results["type_icon"][1].append(results[f"id_icon_{i+1}"][0])
    
    # ---- RT_GROUP_ICON DATA ----
    group_icon_data = bytearray()
    group_icon_data += struct.pack("<HHH", 0, 1, len(icons))
    data_offset_in_group = 6 + 14 * len(icons)
    for icon in icons:
        group_icon_data += struct.pack("<BBBBHHIH",
            icon['width'] if icon['width'] != 256 else 0,
            icon['height'] if icon['height'] != 256 else 0,
            icon['colors'],
            0, icon['planes'], icon['bpp'],
            icon['size'],
            icon['id']
        )
    group_raw_idx = add_obj("RAW", bytes(group_icon_data), "group_icon")
    group_de = struct.pack("<IIII", 0, len(group_icon_data), 1200, 0)
    group_de_idx = add_obj("DATA_ENTRY", group_de, "de_group")
    group_lang_dir = make_simple_dir([(0x0804, False, 0)])
    group_lang_idx = add_obj("DIR", group_lang_dir, "lang_group")
    group_id_dir = make_simple_dir([(1, False, 0)])
    group_id_idx = add_obj("DIR", group_id_dir, "id_group")
    group_type_dir = make_simple_dir([(RT_GROUP_ICON, False, 0)])
    group_type_idx = add_obj("DIR", group_type_dir, "type_group")
    
    # ---- ROOT DIR ----
    root_dir = make_simple_dir([
        (RT_ICON, False, 0),
        (RT_GROUP_ICON, False, 0),
    ])
    root_idx = add_obj("DIR", root_dir, "root")
    
    # ---- Compute offsets and build sections ----
    # $01: DIR objects + DATA_ENTRY objects
    # $02: RAW objects
    
    def fixup_dir(buf_in, offsets_map):
        """Patch directory entries to point to actual offsets."""
        buf = bytearray(buf_in)
        # Directory has: Characteristics(4), TimeDateStamp(4), Major(2), Minor(2), NumNamed(2), NumId(2)
        num_named = struct.unpack_from("<H", buf, 12)[0]
        num_id = struct.unpack_from("<H", buf, 14)[0]
        num_entries = num_named + num_id
        
        entry_start = 16
        for i in range(num_entries):
            pos = entry_start + i * 8
            # Name/ID field
            name_or_id = struct.unpack_from("<I", buf, pos)[0]
            # Offset field
            offset_val = struct.unpack_from("<I", buf, pos + 4)[0]
            if offset_val & 0x80000000:
                # Subdirectory: patch with actual offset
                orig = offset_val & 0x7FFFFFFF
                if orig in offsets_map:
                    new_val = offsets_map[orig] | 0x80000000
                    struct.pack_into("<I", buf, pos + 4, new_val)
            else:
                # Data entry: patch with actual offset
                if offset_val in offsets_map:
                    new_val = offsets_map[offset_val] & 0x7FFFFFFF
                    struct.pack_into("<I", buf, pos + 4, new_val)
        
        return bytes(buf)
    
    # Assign offsets in $01
    section01 = bytearray()
    offsets_01 = {}
    
    for idx, (obj_type, obj_data, name) in enumerate(objects):
        if obj_type in ("DIR", "DATA_ENTRY"):
            offsets_01[idx] = len(section01)
            section01 += obj_data
            # Pad to 4 bytes
            while len(section01) % 4 != 0:
                section01.append(0)
    
    # Assign offsets in $02  
    section02 = bytearray()
    offsets_02 = {}
    
    for idx, (obj_type, obj_data, name) in enumerate(objects):
        if obj_type == "RAW":
            offsets_02[idx] = len(section02)
            section02 += obj_data
            while len(section02) % 4 != 0:
                section02.append(0)
    
    # Combine offsets
    all_offsets = {}
    for idx in offsets_01:
        all_offsets[idx] = offsets_01[idx]
    for idx in offsets_02:
        all_offsets[idx] = len(section01) + offsets_02[idx]  # Global RVA
    
    # Now fixup all directories with correct offsets
    fixed_objects = {}
    for idx, (obj_type, obj_data, name) in enumerate(objects):
        if obj_type == "DIR":
            # Fix up the directory entries
            fixed = fixup_dir(obj_data, all_offsets)
            fixed_objects[idx] = fixed
        elif obj_type == "DATA_ENTRY":
            # Fix up DATA_ENTRY to point to raw data
            # Find the raw data this DE points to
            raw_idx = None
            raw_size = 0
            for de_idx, raw_i, size in icon_de_idxs:
                if de_idx == idx:
                    raw_idx = raw_i
                    raw_size = size
                    break
            if idx == group_de_idx:
                raw_idx = group_raw_idx
                raw_size = len(group_icon_data)
            
            if raw_idx is not None and raw_idx in all_offsets:
                rva = all_offsets[raw_idx]
                de = struct.pack("<IIII", rva, raw_size, 1200, 0)
                fixed_objects[idx] = de
            else:
                fixed_objects[idx] = obj_data
    
    # Rebuild section01 with fixed data
    section01 = bytearray()
    for idx, (obj_type, _, _) in enumerate(objects):
        if obj_type in ("DIR", "DATA_ENTRY") and idx in fixed_objects:
            section01 += fixed_objects[idx]
            while len(section01) % 4 != 0:
                section01.append(0)
    
    # Link directories by patching placeholder values
    # Actually we need to insert proper subdirectory offsets into each directory
    
    # Let me rethink this... This approach is getting very messy.
    # The fundamental issue is that we need to represent a tree structure
    # where nodes reference each other by offset.
    
    # Let me use a flat approach: build all structures sequentially,
    # recording their positions, then go back and fix up all references.
    
    return rebuild_complete(icons, icon_data)


def rebuild_complete(icons, icon_data):
    """Complete rebuild with a simpler sequential approach."""
    
    # This time, we build everything as one flat blob and keep track of positions
    blob = bytearray()
    patches = []  # (offset, value_size, target_name) - patches to apply later
    positions = {}  # name -> offset
    raw_section_split = 0  # offset where $01 ends and $02 begins
    
    def append(data, name):
        nonlocal blob
        while len(blob) % 4 != 0:
            blob.append(0)
        pos = len(blob)
        positions[name] = pos
        blob += data
        return pos
    
    def append_patchable(data, name):
        """Append data and record position for later patching."""
        return append(data, name)
    
    def make_directory(entries_info, name):
        """
        entries_info: list of (id_or_name, is_name, target_name)
        """
        num_named = sum(1 for e in entries_info if e[1])
        num_id = sum(1 for e in entries_info if not e[1])
        
        dir_header = struct.pack("<IIHHHH", 0, 0, 0, 0, num_named, num_id)
        
        entry_data = bytearray()
        string_data = bytearray()
        string_offset = len(dir_header) + 8 * (num_named + num_id)
        
        for name_or_id, is_name, target_name in entries_info:
            if is_name:
                # Named entry - store string
                name_str = name_or_id
                name_bytes = name_str.encode('utf-16-le')
                name_entry = struct.pack("<H", len(name_str)) + name_bytes
                str_off = string_offset + len(string_data)
                entry_data += struct.pack("<I", str_off | 0x80000000)
                string_data += name_entry
            else:
                entry_data += struct.pack("<I", name_or_id)
            
            # Placeholder offset - will be patched
            entry_data += struct.pack("<I", 0)  # placeholder
        
        full_dir = bytearray(dir_header)
        full_dir += entry_data
        full_dir += string_data
        
        pos = append(bytes(full_dir), name)
        
        # Record patches for each entry
        entry_start = len(dir_header)
        for i, (_, is_name, target_name) in enumerate(entries_info):
            patch_pos = pos + entry_start + i * 8 + 4  # offset field position
            patches.append((patch_pos, target_name))
        
        return pos
    
    def make_data_entry(raw_data_name, size):
        de = struct.pack("<IIII", 0, size, 1200, 0)  # RVA placeholder
        pos = append(de, f"de:{raw_data_name}")
        patches.append((pos, f"de:{raw_data_name}"))
        return pos
    
    # Build bottom-up
    
    # Raw icon data
    for i, icon in enumerate(icons):
        append(icon['data'], f"raw_icon_{i+1}")
    
    raw_section_split = len(blob)  # Everything after this is $02
    
    # Data entries for icons
    for i, icon in enumerate(icons):
        make_data_entry(f"raw_icon_{i+1}", icon['size'])
    
    # Language directories for each icon
    for i in range(len(icons)):
        make_directory(
            [(0x0804, False, f"de:raw_icon_{i+1}")],
            f"dir:lang_icon_{i+1}"
        )
    
    # ID directories for each icon
    for i in range(len(icons)):
        make_directory(
            [(icons[i]['id'], False, f"dir:lang_icon_{i+1}")],
            f"dir:id_icon_{i+1}"
        )
    
    # RT_ICON type directory
    icon_type_entries = [(icons[i]['id'], False, f"dir:id_icon_{i+1}") for i in range(len(icons))]
    make_directory(icon_type_entries, "dir:type_icon")
    
    # RT_GROUP_ICON
    group_icon_data = bytearray()
    group_icon_data += struct.pack("<HHH", 0, 1, len(icons))
    for icon in icons:
        group_icon_data += struct.pack("<BBBBHHIH",
            icon['width'] if icon['width'] != 256 else 0,
            icon['height'] if icon['height'] != 256 else 0,
            icon['colors'], 0,
            icon['planes'], icon['bpp'],
            icon['size'], icon['id']
        )
    append(bytes(group_icon_data), "raw_group_icon")
    make_data_entry("raw_group_icon", len(group_icon_data))
    make_directory([(0x0804, False, "de:raw_group_icon")], "dir:lang_group")
    make_directory([(1, False, "dir:lang_group")], "dir:id_group")
    make_directory([(RT_GROUP_ICON, False, "dir:id_group")], "dir:type_group")
    
    # Root
    make_directory([
        (RT_ICON, False, "dir:type_icon"),
        (RT_GROUP_ICON, False, "dir:type_group"),
    ], "dir:root")
    
    # Now apply patches
    blob = bytearray(blob)
    for patch_pos, target_name in patches:
        if target_name not in positions:
            print(f"WARNING: target '{target_name}' not found")
            continue
        target_pos = positions[target_name]
        if target_name.startswith("dir:"):
            # Directory: high bit set
            struct.pack_into("<I", blob, patch_pos, target_pos | 0x80000000)
        else:
            # Data entry: RVA is absolute
            struct.pack_into("<I", blob, patch_pos, target_pos)
    
    # Split: $01 from 0 to raw_section_split, $02 from raw_section_split
    section01 = bytes(blob[:raw_section_split])
    section02 = bytes(blob[raw_section_split:])
    
    return build_coff(section01, section02)


def build_coff(section01, section02):
    """Build final COFF object file."""
    
    num_sections = 2
    file_header_size = 20
    section_header_size = 40
    
    # Section headers
    sh1 = struct.pack("<8sIIIIIIHHI",
        b".rsrc$01",           # Name
        len(section01),        # VirtualSize
        0,                     # VirtualAddress
        len(section01),        # SizeOfRawData
        file_header_size + num_sections * section_header_size,  # PointerToRawData
        0, 0,                  # PointerToRelocations, PointerToLinenumbers
        0, 0,                  # NumberOfRelocations, NumberOfLinenumbers
        IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ | IMAGE_SCN_LNK_INFO | IMAGE_SCN_LNK_REMOVE
    )
    
    sh2 = struct.pack("<8sIIIIIIHHI",
        b".rsrc$02",
        len(section02),
        0,
        len(section02),
        file_header_size + num_sections * section_header_size + len(section01),
        0, 0, 0, 0,
        IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ | IMAGE_SCN_LNK_INFO | IMAGE_SCN_LNK_REMOVE
    )
    
    # File header
    # Note: PointerToSymbolTable and NumberOfSymbols = 0 works
    # But actually Go's linker might need SizeOfOptionalHeader = 0 for objects
    file_header = struct.pack("<HHIIIHH",
        IMAGE_FILE_MACHINE_AMD64,  # Machine (x64)
        num_sections,              # NumberOfSections
        0,                         # TimeDateStamp
        0,                         # PointerToSymbolTable
        0,                         # NumberOfSymbols
        0,                         # SizeOfOptionalHeader (0 for object files)
        0                          # Characteristics (will be 0 for .syso? Actually should have flags)
    )
    
    output = bytearray()
    output += file_header
    output += sh1
    output += sh2
    output += section01
    output += section02
    
    return bytes(output)


def main():
    if not os.path.exists(ICO_PATH):
        print(f"Error: {ICO_PATH} not found. Run in yunfei/ directory.")
        return
    
    with open(ICO_PATH, 'rb') as f:
        icon_data = f.read()
    
    print(f"ICO file size: {len(icon_data)} bytes")
    
    syso_data = rebuild_complete(None, icon_data)
    
    with open(OUT_PATH, 'wb') as f:
        f.write(syso_data)
    
    print(f"Generated {OUT_PATH} ({len(syso_data)} bytes)")


if __name__ == '__main__':
    # Read ICO and parse icons
    with open(ICO_PATH, 'rb') as f:
        icon_data = f.read()
    
    # Parse ICO
    reserved, img_type, count = struct.unpack_from("<HHH", icon_data, 0)
    
    icons = []
    for i in range(count):
        off = 6 + i * 16
        w, h, colors, reserved2, planes, bpp, size, data_offset = struct.unpack_from(
            "<BBBBHHII", icon_data, off
        )
        if w == 0: w = 256
        if h == 0: h = 256
        icons.append({
            'width': w, 'height': h, 'colors': colors,
            'planes': planes, 'bpp': bpp, 'size': size,
            'offset': data_offset,
            'data': icon_data[data_offset:data_offset + size],
            'id': i + 1
        })
    
    print(f"ICO contains {count} icon(s)")
    for ic in icons:
        print(f"  {ic['width']}x{ic['height']} @ {ic['bpp']}bpp, {ic['size']} bytes")
    
    syso_data = rebuild_complete(icons, icon_data)
    
    with open(OUT_PATH, 'wb') as f:
        f.write(syso_data)
    
    print(f"Generated {OUT_PATH} ({len(syso_data)} bytes)")
