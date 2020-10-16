import glob
import sys
import random
import numpy as np
from PIL import Image
import tensorflow as tf
from tensorflow.keras.preprocessing.image import img_to_array
from tensorflow.keras.preprocessing.image import load_img

# this is the fastest way to get vectors from inputs, but should be an independent module to call as a library if used in production
model = tf.keras.applications.ResNet50(include_top=False, weights='imagenet', input_shape=(224, 224, 3), pooling='avg')


# add user input and dataset file paths into variables to call later
image_collection = glob.glob('DATA/*/*.jpg')
input_file = sys.argv[1]

def generate_values(files):
    '''take file collection paths, and return dict as file: vector'''
    table = {}
    for file in files:
        image = load_img(file, target_size=(224, 224))
        numpy_image = img_to_array(image)
        input_image = np.expand_dims(numpy_image, axis=0)
        input_vector = model.predict(input_image)
        table[file] = input_vector
    return table

def anchor_value(file):
    '''take singe file from user input and return file:vector'''
    image = load_img(file, target_size=(224, 224))
    numpy_image = img_to_array(image)
    input_image = np.expand_dims(numpy_image, axis=0)
    input_vector = model.predict(input_image)

    return {file: input_vector}

anchor_value = anchor_value(input_file)
table = generate_values(image_collection)

def triplet_loss(anchor, input1, input2):
    '''input 3 files: user_input, random1, random2 from file collection and return file name for the lower loss'''
    first_input = ((anchor_value[anchor] - table[input1]) ** 2).mean(axis=1)
    second_input = ((anchor_value[anchor] - table[input2]) ** 2).mean(axis=1)

    if first_input > second_input:
        return input2
    elif second_input > first_input:
        return input1

def binary_search(image_list, anchor):
    '''take a list of file paths, and return half of the list that is closer to the anchor file'''
    if len(image_list) % 2 != 0:
        image_list.pop()
    
    positive_list = []

    for i in range(0, int(len(image_list)/2), 2):
        image1 = image_list[i]
        image2 = image_list[i+1]

        positive_list.append(triplet_loss(anchor, image1, image2))
    return positive_list
    
def hierarchical_search(image_list, anchor):
    '''input list of file paths, runs binary search recursively until only one file is left'''
    while len(image_list) != 1:
        image_list = binary_search(image_list, anchor)
    return image_list

result = hierarchical_search(image_collection, input_file)

'''
TODO: return result to the network layer once lookup is done the the closest file has been found
'''



# show the results for demo
im1 = Image.open(input_file)
im2 = Image.open(result[0])

im1 = im1.resize((224, 224))
im2 = im2.resize((224, 224))

Image.fromarray(np.hstack((np.array(im2),np.array(im1)))).show()