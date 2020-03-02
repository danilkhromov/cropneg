--[[
    Automatic negative cropping for darktable

    Copyright (c) 2020 Daniil Khromov

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation; either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program. If not, see <http://www.gnu.org/licenses/>.
]]
--[[
    Automatic negative cropping

    This script automatically crops negative scans that were imported into darktable.
    Cropping works best with scans from Epson flatbed scanner and with minimal cropping area.

    ADDITIONAL SOFTWARE NEEDED FOR THIS SCRIPT
    * https://github.com/danilkhromov/cropneg

    USAGE
    * require this script from your main luarc file
    * before export click on "enable auto crop" under the export tab
    * export images

    CAVEATS
    * script works best with well exposed frames with clearly defined frame borders
    * if the image cannot be cropped it will be saved in original size (no cropping)
]]

local darktable = require "darktable"

local crop_command = "~/bin/cropneg -f "

local enable_auto_crop = darktable.new_widget("check_button") {
    label = "enable auto crop"
}

local crop_widget = darktable.new_widget("box") {
    orientation = horizontal,
    enable_auto_crop
}

darktable.register_lib("control_auto_crop_ui", "auto crop image on export", true, false, {
    [darktable.gui.views.lighttable] = { "DT_UI_CONTAINER_PANEL_RIGHT_CENTER", 0 }
}, crop_widget
);

darktable.register_event("intermediate-export-image",
        function(event, image, filename, format, storage)

            if not enable_auto_crop.value == true then
                return
            end

            os.execute(crop_command .. image.path .. "/" .. image.filename .. " -n " .. filename)
        end
)