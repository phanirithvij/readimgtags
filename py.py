from PIL import Image, ExifTags
import sys
import base64

img = Image.open(sys.argv[1])
exif = { ExifTags.TAGS[k]: v for k, v in img._getexif().items() if k in ExifTags.TAGS }
# print(exif['XPKeywords'])
# print(exif['XPComment'])
print(exif['ImageDescription'])
print(exif['UserComment'][8:])
