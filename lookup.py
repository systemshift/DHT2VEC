import glob
import sys
import random
import numpy as np
from PIL import Image
import tensorflow as tf
from tensorflow.keras.preprocessing.image import img_to_array
from tensorflow.keras.preprocessing.image import load_img


model = tf.keras.applications.ResNet50(include_top=False, weights='imagenet', input_shape=(224, 224, 3), pooling='avg')

image_collection = glob.glob('DATA/*/*.jpg')
input_file = sys.argv[1]

def generate_values(files):
    table = {}
    for file in files:
        image = load_img(file, target_size=(224, 224))
        numpy_image = img_to_array(image)
        input_image = np.expand_dims(numpy_image, axis=0)
        input_vector = model.predict(input_image)
        table[file] = input_vector
    return table

def anchor_value(file):
    image = load_img(file, target_size=(224, 224))
    numpy_image = img_to_array(image)
    input_image = np.expand_dims(numpy_image, axis=0)
    input_vector = model.predict(input_image)

    return {file: input_vector}

anchor_value = anchor_value(input_file)
table = generate_values(image_collection)

def triplet_loss(anchor, input1, input2):
    first_input = ((anchor_value[anchor] - table[input1]) ** 2).mean(axis=1)
    second_input = ((anchor_value[anchor] - table[input2]) ** 2).mean(axis=1)

    if first_input > second_input:
        return input2
    elif second_input > first_input:
        return input1

def binary_search(image_list, anchor):
    if len(image_list) % 2 != 0:
        image_list.pop()
    
    positive_list = []

    for i in range(0, int(len(image_list)/2), 2):
        image1 = image_list[i]
        image2 = image_list[i+1]

        positive_list.append(triplet_loss(anchor, image1, image2))
    return positive_list
    
def hierarchical_search(image_list, anchor):
    while len(image_list) != 1:
        image_list = binary_search(image_list, anchor)
    return image_list

result = hierarchical_search(image_collection, input_file)


# show the results
im1 = Image.open(input_file)
im2 = Image.open(result[0])

Image.fromarray(np.hstack((np.array(im1),np.array(im2)))).show()