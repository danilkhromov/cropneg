---
--- Generated by EmmyLua(https://github.com/EmmyLua)
--- Created by danil.
--- DateTime: 2/21/20 7:23 PM
---

local darktable = require "darktable"

function crop_negative_event(event, image)
    crop_negative(image)
end

local path

function crop_negative(image)

    path = image.path .. "/cropped"

    os.execute("mkdir -p " .. image.path .. "/cropped")

    print("~/projects/cropneg/cropneg --file " .. image.path .. "/" .. image.filename)
    os.execute("~/projects/cropneg/cropneg --file " .. image.path .. "/" .. image.filename)

    return 1
end

function apply_negative_crop(shortcut)

    local images = darktable.gui.action_images
    local images_processed = 0
    local images_submitted = 0

    for _, image in pairs(images) do
        images_submitted = images_submitted + 1
        images_processed = images_processed + crop_negative(image)
    end

    darktable.print("Cropped " .. images_processed .. " out of " .. images_submitted .. " image(s)")

    if images_processed > 0 then
        print("mv *cropped_* " .. path .. "/cropped")
        os.execute("mv cropped_* " .. path .. "/cropped")
    end
end

darktable.register_event("shortcut", apply_negative_crop, "Automatically crop selected negative scans")