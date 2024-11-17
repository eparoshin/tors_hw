import math

with open('output.txt', 'w') as f:
    num_steps = int((1000.0 - 0.0) / 0.001) + 1
    for i in range(num_steps):
        x = i * 0.001
        y = math.cos(x) ** 2
        f.write(f"{x} {y}\n")
