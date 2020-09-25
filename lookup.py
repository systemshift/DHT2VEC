import numpy as np
import tensorflow as tf
from tensorflow.keras.preprocessing.image import img_to_array
from tensorflow.keras.preprocessing.image import load_img


model = tf.keras.applications.ResNet50(include_top=False, weights='imagenet', input_shape=(224, 224, 3), pooling='avg')


image = load_img('DATA/airship/1276200263_900234bb0a.jpg', target_size=(224, 224))
numpy_image = img_to_array(image)
input_image = np.expand_dims(numpy_image, axis=0)
input_vector = model.predict(input_image)


image2 = load_img('DATA/airship/2294241647_34d8ac7831.jpg', target_size=(224, 224))
numpy_image2 = img_to_array(image2)
input_image2 = np.expand_dims(numpy_image2, axis=0)
input_vector2 = model.predict(input_image2)


image3 = load_img('DATA/alder/1283777590_80e34a0420.jpg', target_size=(224, 224))
numpy_image3 = img_to_array(image3)
input_image3 = np.expand_dims(numpy_image3, axis=0)
input_vector3 = model.predict(input_image3)

loss = tf.keras.losses.MSE(input_vector, input_vector2)
loss2 = tf.keras.losses.MSE(input_vector, input_vector3)


print('loss1:' + str(loss.numpy()[0]))
print('loss2:' + str(loss2.numpy()[0]))