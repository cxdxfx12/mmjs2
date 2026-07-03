import struct
import sys

def generate_syso(ico_path, syso_path, arch='386'):
    MACHINE_386 = 0x014c
    MACHINE_AMD64 = 0x8664
    machine = MACHINE_AMD64 if arch == 'amd64' else MACHINE_386
    
    with open(ico_path, 'rb') as f:
        ico = f.read()
    
    # Parse ICO header
    if ico[0:2] != b'\x00\x00' or ico[2:4] != b'\x01\x00':
        raise ValueError("Not a valid ICO file")
    
    num_images = struct.unpack_from('<H', ico, 4)[0]
    
    # Extract images
    images = []
    for i in range(num_images):
        off = 6 + 16 * i
        w = ico[off] or 256
        h = ico[off + 1] or 256
        colors = ico[off + 2]
        planes = struct.unpack_from('<H', ico, off + 4)[0]
        bpp = struct.unpack_from('<H', ico, off + 6)[0]
        size = struct.unpack_from('<I', ico, off + 8)[0]
        img_off = struct.unpack_from('<I', ico, off + 12)[0]
        images.append({
            'data': ico[img_off:img_off + size],
            'size': size,
            'w': w, 'h': h, 'colors': colors,
            'planes': planes, 'bpp': bpp,
        })
    
    # Build resource data (icon images + group icon)
    # We need a proper IMAGE_RESOURCE_DIRECTORY structure
    
    # Resource layout:
    # [Root Dir] -> [Type Dir: RT_GROUP_ICON, RT_ICON] -> [Name Dir: ID 1] -> [Lang Dir: 0x409] -> [Data Entry] -> [actual data]
    
    # Calculate offsets
    # Root directory: 16 + 2*8 = 32
    # RT_GROUP_ICON subdir: 16 + 8 = 24
    # RT_GROUP_ICON lang dir: 16 + 8 = 24
    # RT_GROUP_ICON data entry: 16
    # RT_ICON subdir: 16 + num_images*8 = 16 + num_images*8
    # RT_ICON lang dirs: num_images * (16 + 8) = num_images * 24
    # RT_ICON data entries: num_images * 16
    # Total header size: 32 + 24+24+16 + (16+num_images*8) + num_images*24 + num_images*16
    
    root_size = 32  # 16 + 2*8
    
    # RT_GROUP_ICON subtree
    group_name_size = 24  # 16 + 1*8
    group_lang_size = 24  # 16 + 1*8  
    group_data_entry_size = 16
    
    # RT_ICON subtree
    icon_name_size = 16 + num_images * 8
    icon_lang_size = num_images * 24  # each has 16+8
    icon_data_entry_size = num_images * 16
    
    header_end = (root_size + 
                  group_name_size + group_lang_size + group_data_entry_size +
                  icon_name_size + icon_lang_size + icon_data_entry_size)
    
    # Data section starts after header (aligned to 16 bytes boundary for resources)
    data_start = header_end
    # Align to 8 bytes (typical)
    while data_start % 8 != 0:
        data_start += 1
    
    # Build group icon data (GRPICONDIR)
    group_data = bytearray()
    group_data += struct.pack('<HHH', 0, 1, num_images)
    for i, img in enumerate(images):
        group_data += struct.pack('<BBBBHHHIH',
            img['w'] if img['w'] < 256 else 0,
            img['h'] if img['h'] < 256 else 0,
            img['colors'] if img['colors'] < 256 else 0,
            0,
            img['planes'],
            img['bpp'],
            img['size'],
            img['size'],
            i + 1
        )
    
    # Calculate data offsets
    group_rva = data_start
    icon_rvas = []
    current = data_start + len(group_data)
    for img in images:
        icon_rvas.append(current)
        current += img['size']
    
    total_rsrc_size = current
    
    # Now build the .rsrc section data
    rsrc = bytearray()
    
    # === Root directory ===
    # Entries sorted by ID: RT_ICON (3), RT_GROUP_ICON (14)
    rsrc += struct.pack('<IIHHHH', 0, 0, 0, 0, 0, 2)  # header: 2 ID entries
    
    # Root entry 0: RT_ICON (ID 3) -> subdirectory
    rt_icon_name_off = root_size + group_name_size + group_lang_size + group_data_entry_size
    rsrc += struct.pack('<II', 0x80000000 | 3, 0x80000000 | rt_icon_name_off)
    
    # Root entry 1: RT_GROUP_ICON (ID 14) -> subdirectory
    rt_group_name_off = root_size
    rsrc += struct.pack('<II', 0x80000000 | 14, 0x80000000 | rt_group_name_off)
    
    assert len(rsrc) == root_size
    
    # === RT_GROUP_ICON name directory (ID=1) ===
    rsrc += struct.pack('<IIHHHH', 0, 0, 0, 0, 0, 1)  # 1 ID entry
    # Entry: ID 1 -> lang subdirectory
    group_lang_off = rt_group_name_off + group_name_size
    rsrc += struct.pack('<II', 0x80000000 | 1, 0x80000000 | group_lang_off)
    
    assert len(rsrc) == rt_group_name_off + group_name_size
    
    # === RT_GROUP_ICON lang directory (LANG=0x0409) ===
    rsrc += struct.pack('<IIHHHH', 0, 0, 0, 0, 0, 1)
    # Entry: 0x0409 -> data entry
    group_data_entry_off = group_lang_off + group_lang_size
    rsrc += struct.pack('<II', 0x80000000 | 0x0409, group_data_entry_off)
    
    assert len(rsrc) == group_lang_off + group_lang_size
    
    # === RT_GROUP_ICON data entry ===
    rsrc += struct.pack('<IIII', group_rva, len(group_data), 0, 0)
    
    assert len(rsrc) == group_data_entry_off + group_data_entry_size
    
    # === RT_ICON name directory (IDs 1..num_images) ===
    assert len(rsrc) == rt_icon_name_off
    rsrc += struct.pack('<IIHHHH', 0, 0, 0, 0, 0, num_images)
    
    icon_lang_base = rt_icon_name_off + icon_name_size
    icon_data_entry_base = icon_lang_base + icon_lang_size
    
    for i in range(num_images):
        # Entry: ID (i+1) -> lang subdirectory
        lang_off = icon_lang_base + i * 24
        rsrc += struct.pack('<II', 0x80000000 | (i + 1), 0x80000000 | lang_off)
    
    assert len(rsrc) == rt_icon_name_off + icon_name_size
    
    # === RT_ICON lang directories ===
    for i in range(num_images):
        rsrc += struct.pack('<IIHHHH', 0, 0, 0, 0, 0, 1)
        # Entry: 0x0409 -> data entry
        data_entry_off = icon_data_entry_base + i * 16
        rsrc += struct.pack('<II', 0x80000000 | 0x0409, data_entry_off)
    
    assert len(rsrc) == icon_lang_base + icon_lang_size
    
    # === RT_ICON data entries ===
    for i in range(num_images):
        rsrc += struct.pack('<IIII', icon_rvas[i], images[i]['size'], 0, 0)
    
    assert len(rsrc) == icon_data_entry_base + icon_data_entry_size
    
    # Pad to data_start
    while len(rsrc) < data_start:
        rsrc += b'\x00'
    
    # === Actual resource data ===
    rsrc += group_data
    for img in images:
        rsrc += img['data']
    
    # Pad to 4 bytes
    while len(rsrc) % 4 != 0:
        rsrc += b'\x00'
    
    # === Build COFF file ===
    coff = bytearray()
    
    # COFF header
    coff += struct.pack('<HHIIIHH',
        machine,   # Machine
        1,         # NumberOfSections
        0,         # TimeDateStamp
        0,         # PointerToSymbolTable
        0,         # NumberOfSymbols
        0,         # SizeOfOptionalHeader
        0x0002,    # Characteristics: executable image
    )
    
    # Section header: .rsrc
    section_name = b'.rsrc\x00\x00\x00'
    raw_data_offset = 20 + 40  # COFF header + 1 section header
    # Align raw data
    if raw_data_offset % 8 != 0:
        raw_data_offset += 8 - raw_data_offset % 8
    
    coff += section_name
    coff += struct.pack('<IIIIIIHHI',
        len(rsrc),      # VirtualSize
        0,              # VirtualAddress
        len(rsrc),      # SizeOfRawData
        raw_data_offset, # PointerToRawData
        0,              # PointerToRelocations
        0,              # PointerToLinenumbers
        0,              # NumberOfRelocations
        0,              # NumberOfLinenumbers
        0x40000040,     # Characteristics: read | data
    )
    
    # Pad to raw data offset
    while len(coff) < raw_data_offset:
        coff += b'\x00'
    
    # Section data
    coff += rsrc
    
    with open(syso_path, 'wb') as f:
        f.write(bytes(coff))
    
    print(f"Generated {syso_path}: {len(coff)} bytes, {len(images)} icon images")

if __name__ == '__main__':
    if len(sys.argv) < 3:
        print(f"Usage: {sys.argv[0]} input.ico output.syso [386|amd64]")
        sys.exit(1)
    arch = sys.argv[3] if len(sys.argv) > 3 else '386'
    generate_syso(sys.argv[1], sys.argv[2], arch)
