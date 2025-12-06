
import cv2
import numpy as np
from PIL import Image, ImageDraw, ImageColor
import random
import argparse
import heapq

# --- Configuration ---
WIDTH, HEIGHT = 1200, 630
GRID_SPACING = 60
MIN_WIRES = 0
MAX_WIRES = 4
MIN_COMPONENTS = 15
MAX_COMPONENTS = 40
COMPONENT_MIN_SIZE = 15
COMPONENT_MAX_SIZE = 100

# --- Color Palettes ---
PALETTES = [
    ["#d8f3dc", "#b7e4c7", "#95d5b2", "#74c69d", "#52b788", "#40916c", "#2d6a4f"], # Green
    ["#fde2e4", "#fad2e1", "#fbc3d4", "#f9b4c8", "#f8a5bc", "#f796b0", "#f687a3"], # Pink/Red
    ["#ADD8E6", "#87CEEB", "#6495ED", "#4169E1", "#1E90FF"], # Blue
    ["#F2E7FE", "#E6CCFB", "#D1ACF6", "#BB8CEF", "#A36EE8"], # Purple
    ["#FFFAD3", "#FFECB3", "#FFDD88", "#FFCE5A", "#FFBF2B"], # Yellow/Orange
    ["#FFCDD2", "#EF9A9A", "#E57373", "#EF5350", "#F44336"], # Red
    ["#F5F5F5", "#E0E0E0", "#BDBDBD", "#9E9E9E", "#757575"], # Grey
    ["#D7CCC8", "#BCAAA4", "#A1887F", "#8D6E63", "#795548"], # Brown
    ["#E0F7FA", "#B2EBF2", "#80DEEA", "#4DD0E1", "#00BCD4"], # Cyan
    ["#C5CAE9", "#9FA8DA", "#7986CB", "#5C6BC0", "#3F51B5"], # Indigo/Blue
    ["#A8DADC", "#83C5BE", "#6D9F9D", "#548A85", "#3C756F"], # Teal
]

def _get_hexagon_vertices(center, size):
    """Calculates the 6 vertices of a regular hexagon."""
    vertices = []
    for i in range(6):
        angle_deg = 60 * i - 30 # -30 to make it flat top
        angle_rad = np.pi / 180 * angle_deg
        x = center[0] + size * np.cos(angle_rad)
        y = center[1] + size * np.sin(angle_rad)
        vertices.append((x, y))
    return vertices

def _get_star_vertices(center, size, points=5):
    """Calculates the vertices of a star."""
    vertices = []
    inner_radius = size * 0.4
    for i in range(points * 2):
        r = size if i % 2 == 0 else inner_radius
        angle_deg = (360 / (points * 2)) * i - 90
        angle_rad = np.pi / 180 * angle_deg
        x = center[0] + r * np.cos(angle_rad)
        y = center[1] + r * np.sin(angle_rad)
        vertices.append((x, y))
    return vertices


def generate_art(output_path, style='grid', seed=None, add_network=False):
    if seed is not None:
        random.seed(seed)

    # 1. --- Setup ---
    background_colors = ["#111111", "#0D1B2A", "#1B263B", "#22223B", "#0A0A14", "#201E1F"]
    bg_color = random.choice(background_colors)
    img = Image.new("RGB", (WIDTH, HEIGHT), color=bg_color)

    # Draw background grid
    try:
        bg_color_rgb = ImageColor.getrgb(bg_color)
        grid_color = tuple(min(255, c + 20) for c in bg_color_rgb)
        grid_draw = ImageDraw.Draw(img)
        for x in range(0, WIDTH, GRID_SPACING):
            grid_draw.line([(x, 0), (x, HEIGHT)], fill=grid_color, width=1)
        for y in range(0, HEIGHT, GRID_SPACING):
            grid_draw.line([(0, y), (WIDTH, y)], fill=grid_color, width=1)
    except Exception:
        pass # Fail silently if color parsing fails

    draw = ImageDraw.Draw(img)
    palette = random.choice(PALETTES)

    # 2. --- Create Grid & Points ---
    jitter_amount = 0 if style == 'grid' else 5
    grid_points = []
    for x in range(0, WIDTH + GRID_SPACING, GRID_SPACING):
        for y in range(0, HEIGHT + GRID_SPACING, GRID_SPACING):
            jitter_x = random.randint(-jitter_amount, jitter_amount)
            jitter_y = random.randint(-jitter_amount, jitter_amount)
            grid_points.append((x + jitter_x, y + jitter_y))

    # Determine Component Points
    num_components = random.randint(MIN_COMPONENTS, MAX_COMPONENTS)
    component_points = []
    for _ in range(num_components):
        component_points.append(random.choice(grid_points))

    # 3. --- Draw Wires based on Style ---
    num_wires = random.randint(MIN_WIRES, MAX_WIRES)

    if style == 'radial':
        hub = (WIDTH / 2 + random.uniform(-WIDTH/4, WIDTH/4), HEIGHT / 2 + random.uniform(-HEIGHT/4, HEIGHT/4))
        for _ in range(num_wires):
            start_point = hub
            end_point = random.choice(grid_points)
            draw.line([start_point, end_point], fill=random.choice(palette), width=random.choice([1, 1, 2]))
    
    elif style == 'flow':
        for _ in range(num_wires):
            start_point = random.choice([p for p in grid_points if p[0] < WIDTH / 2])
            end_point = random.choice([p for p in grid_points if p[0] > start_point[0] + GRID_SPACING])
            draw.line([start_point, end_point], fill=random.choice(palette), width=random.choice([1, 1, 2]))

    else: # 'grid' or 'random'
        for _ in range(num_wires):
            start_point = random.choice(grid_points)
            end_point = random.choice(grid_points)
            color = random.choice(palette)
            width = random.choice([1, 1, 2])
            
            # In 'grid' mode, always draw right-angled wires. In 'random' mode, 20% chance.
            if style == 'grid' or random.random() < 0.2:
                mid_point_x = (start_point[0], end_point[1])
                mid_point_y = (end_point[0], start_point[1])
                mid_point = random.choice([mid_point_x, mid_point_y])
                draw.line([start_point, mid_point, end_point], fill=color, width=width)
            else:
                draw.line([start_point, end_point], fill=color, width=width)

    # 4. --- Draw Network Overlay ---
    if add_network and len(component_points) > 1:
        # Connect each component to its N nearest neighbors
        for i, p1 in enumerate(component_points):
            distances = []
            for j, p2 in enumerate(component_points):
                if i == j: continue
                dist = np.sqrt((p1[0] - p2[0])**2 + (p1[1] - p2[1])**2)
                distances.append((dist, p2))

            num_neighbors = random.randint(0, 4)
            if len(distances) < num_neighbors:
                num_neighbors = len(distances)
                
            neighbors = heapq.nsmallest(num_neighbors, distances)

            for dist, neighbor_point in neighbors:
                if dist < WIDTH / 3.5: # Limit connection distance
                    draw.line([p1, neighbor_point], fill=random.choice(palette), width=random.randint(8, 15))

    # 5. --- Draw Components ---
    for point in component_points: # Iterate over pre-generated points
        size = random.randint(COMPONENT_MIN_SIZE, COMPONENT_MAX_SIZE)
        color = random.choice(palette)

        shape_choice = random.random()
        if shape_choice < 0.15:
            # Rectangle
            draw.rectangle([point[0] - size, point[1] - size, point[0] + size, point[1] + size], fill=color)
        elif shape_choice < 0.30:
            # Ellipse
            draw.ellipse([point[0] - size, point[1] - size, point[0] + size, point[1] + size], fill=color)
        elif shape_choice < 0.45:
            # Pie Slice
            start = random.randint(0, 360)
            end = start + random.randint(45, 300)
            draw.pieslice([point[0] - size, point[1] - size, point[0] + size, point[1] + size], start, end, fill=color)
        elif shape_choice < 0.60:
            # Hexagon
            draw.polygon(_get_hexagon_vertices(point, size), fill=color)
        elif shape_choice < 0.70:
            # Star
            draw.polygon(_get_star_vertices(point, size, points=random.choice([5,6,7])), fill=color)
        elif shape_choice < 0.85:
            # Right Triangle
            p1 = point
            quadrant = random.randint(1,4)
            if quadrant == 1:
                p2 = (point[0] + size, point[1])
                p3 = (point[0], point[1] + size)
            elif quadrant == 2:
                p2 = (point[0] - size, point[1])
                p3 = (point[0], point[1] + size)
            elif quadrant == 3:
                p2 = (point[0] - size, point[1])
                p3 = (point[0], point[1] - size)
            else:
                p2 = (point[0] + size, point[1])
                p3 = (point[0], point[1] - size)
            draw.polygon([p1, p2, p3], fill=color)
        elif shape_choice < 0.95:
            # Parabola
            parabola_points = []
            for x_local in range(-size, size + 1):
                y_local = int(((x_local / size) ** 2) * size)
                if random.random() < 0.5:
                    parabola_points.append((point[0] + x_local, point[1] - size + y_local))
                else:
                    parabola_points.append((point[0] - size + y_local, point[1] + x_local))
            draw.line(parabola_points, fill=color, width=random.choice([1, 2, 2, 3]))
        else:
            # Original Random Triangle
            p1 = (point[0] + random.randint(-size, size), point[1] + random.randint(-size, size))
            p2 = (point[0] + random.randint(-size, size), point[1] + random.randint(-size, size))
            p3 = (point[0] + random.randint(-size, size), point[1] + random.randint(-size, size))
            draw.polygon([p1, p2, p3], fill=color)

    # 6. --- Post-processing with OpenCV ---
    frame = np.array(img)
    frame = cv2.cvtColor(frame, cv2.COLOR_RGB2BGR)
    frame = cv2.GaussianBlur(frame, (5, 5), 0) # Soften the digital look
    
    # Add a subtle vignette effect
    kernel_x = cv2.getGaussianKernel(WIDTH, 250)
    kernel_y = cv2.getGaussianKernel(HEIGHT, 250)
    kernel = kernel_y * kernel_x.T
    mask = 255 * kernel / np.linalg.norm(kernel)
    vignette = np.copy(frame)
    for i in range(3):
        vignette[:, :, i] = vignette[:, :, i] * mask
    
    final_frame = cv2.addWeighted(frame, 0.7, vignette, 0.3, 0)

    # 6. --- Save the final image ---
    cv2.imwrite(output_path, final_frame)
    print(f"Art saved to {output_path}")

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Generate technical cover art for a blog.")
    parser.add_argument("output", type=str, help="Path to save the output image file (e.g., cover.png).")
    parser.add_argument("--style", type=str, default="grid", choices=['grid', 'radial', 'flow', 'random'], help="The generation style.")
    parser.add_argument("--seed", type=int, default=None, help="Optional random seed for reproducible results.")
    parser.add_argument("--network", action="store_true", help="Add a network graph overlay connecting components.")
    args = parser.parse_args()
    generate_art(args.output, args.style, args.seed, args.network)
