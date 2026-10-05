import random
import string
import sys

# This script will be run with:
# 1st argument as the machineNum
# 2nd ... args as the phrase

# This script will generate a file with randomized lines, and replace certain sections with the given phrase

FindStr = " ".join(sys.argv[2:])
FindStrLen = 80 - len(FindStr)

# generate a random string of a given length
def gen_random_string(length=FindStrLen):
    return ''.join(random.choices(string.ascii_letters + string.digits, k=length))

# generate a random string with the phrase included inside of it
def random_string_with_word():
    rand_str = gen_random_string(FindStrLen) 
    insertion_index = random.randint(0, FindStrLen)  
    return rand_str[:insertion_index] + FindStr + rand_str[insertion_index:] + "\n"

# repeat process for an artificial file's worth of data
line_random_placement = lambda: random_string_with_word()
random_line = lambda: gen_random_string(80) + "\n"
lines = [line_random_placement() for _ in range(50)] + [random_line() for _ in range(50)]
random.shuffle(lines)  

# write resulting logs to a file called "machine.i.log"
file_name = "../machine." + sys.argv[1] + ".log"
with open(file_name, "w") as file:
    file.writelines(lines)

