// Scene setup
const scene = new THREE.Scene();
scene.background = new THREE.Color(0x0a0a0a);
const camera = new THREE.PerspectiveCamera(75, window.innerWidth / window.innerHeight, 0.1, 1000);
camera.position.z = 5;

const renderer = new THREE.WebGLRenderer({ antialias: true });
renderer.setSize(window.innerWidth, window.innerHeight);
renderer.setPixelRatio(window.devicePixelRatio);
renderer.toneMapping = THREE.ReinhardToneMapping;
renderer.toneMappingExposure = 1.2;
document.body.appendChild(renderer.domElement);



// Controls
const controls = new THREE.OrbitControls(camera, renderer.domElement);
controls.enableDamping = true;
controls.dampingFactor = 0.05;
controls.autoRotate = true;
controls.autoRotateSpeed = 0.8;

// Mouse interaction tracking
const mouse = new THREE.Vector2(9999, 9999);
const mouseNDC = new THREE.Vector2(9999, 9999);
const raycaster = new THREE.Raycaster();
let mouseWorld = new THREE.Vector3();
const mouseInfluenceRadius = 1.2;
const mouseRepelStrength = 0.4;

// Smooth mouse-driven orbit offset
const mouseOrbitTarget = { x: 0, y: 0 };
const mouseOrbitCurrent = { x: 0, y: 0 };
const mouseOrbitStrength = 0.3; // max radians of tilt
const mouseOrbitSmoothing = 0.04;

// Mouse press scale effect
let mousePressed = false;
let sphereScaleCurrent = 1.0;
const sphereScalePressed = 1.25;
const sphereScaleSmoothing = 0.08;

window.addEventListener('mousedown', () => { mousePressed = true; });
window.addEventListener('mouseup', () => { mousePressed = false; });
window.addEventListener('touchstart', () => { mousePressed = true; });
window.addEventListener('touchend', () => { mousePressed = false; });

window.addEventListener('mousemove', (e) => {
  mouseNDC.x = (e.clientX / window.innerWidth) * 2 - 1;
  mouseNDC.y = -(e.clientY / window.innerHeight) * 2 + 1;
  mouseOrbitTarget.x = mouseNDC.y * mouseOrbitStrength;
  mouseOrbitTarget.y = mouseNDC.x * mouseOrbitStrength;
});

window.addEventListener('mouseleave', () => {
  mouseNDC.set(9999, 9999);
  mouseOrbitTarget.x = 0;
  mouseOrbitTarget.y = 0;
});

// Particle sphere
const particleCount = 15000;
const radius = 1.5;

const geometry = new THREE.BufferGeometry();
const positions = new Float32Array(particleCount * 3);
const colors = new Float32Array(particleCount * 3);
const sizes = new Float32Array(particleCount);
const originalPositions = new Float32Array(particleCount * 3);

for (let i = 0; i < particleCount; i++) {
  // Fibonacci sphere distribution for even spacing
  const phi = Math.acos(1 - 2 * (i + 0.5) / particleCount);
  const theta = Math.PI * (1 + Math.sqrt(5)) * i;

  const x = radius * Math.sin(phi) * Math.cos(theta);
  const y = radius * Math.sin(phi) * Math.sin(theta);
  const z = radius * Math.cos(phi);

  positions[i * 3] = x;
  positions[i * 3 + 1] = y;
  positions[i * 3 + 2] = z;

  originalPositions[i * 3] = x;
  originalPositions[i * 3 + 1] = y;
  originalPositions[i * 3 + 2] = z;

    // Color gradient from warm amber to rose-pink based on height
    const t = (y / radius + 1) * 0.5;
    colors[i * 3] = 0.95 - t * 0.3;      // R
    colors[i * 3 + 1] = 0.4 + t * 0.15;  // G
    colors[i * 3 + 2] = 0.2 + t * 0.5;   // B

    // Store base colors for ripple effect
    if (!window._baseColors) window._baseColors = new Float32Array(particleCount * 3);
    window._baseColors[i * 3] = colors[i * 3];
    window._baseColors[i * 3 + 1] = colors[i * 3 + 1];
    window._baseColors[i * 3 + 2] = colors[i * 3 + 2];

  sizes[i] = Math.random() * 2.0 + 0.5;
}

geometry.setAttribute('position', new THREE.BufferAttribute(positions, 3));
geometry.setAttribute('color', new THREE.BufferAttribute(colors, 3));
geometry.setAttribute('size', new THREE.BufferAttribute(sizes, 1));

// Custom shader material for soft glowing particles
const vertexShader = `
  attribute float size;
  varying vec3 vColor;
  varying float vDist;
  void main() {
    vColor = color;
    vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
    vDist = -mvPosition.z;
    gl_PointSize = size * (80.0 / -mvPosition.z);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

const fragmentShader = `
  varying vec3 vColor;
  varying float vDist;
  void main() {
    float d = length(gl_PointCoord - vec2(0.5));
    if (d > 0.5) discard;
    float alpha = 1.0 - smoothstep(0.1, 0.5, d);
    alpha *= 0.55;
    float glow = exp(-d * 6.0) * 0.3;
    vec3 col = vColor * 0.8 + glow;
    gl_FragColor = vec4(col, alpha);
  }
`;

const material = new THREE.ShaderMaterial({
  vertexShader,
  fragmentShader,
  vertexColors: true,
  transparent: true,
  depthWrite: false,
  blending: THREE.AdditiveBlending,
});

const particles = new THREE.Points(geometry, material);
scene.add(particles);

// Background star field
const starCount = 8000;
const starGeometry = new THREE.BufferGeometry();
const starPositions = new Float32Array(starCount * 3);
const starColors = new Float32Array(starCount * 3);
const starSizes = new Float32Array(starCount);

for (let i = 0; i < starCount; i++) {
  // Distribute stars in a large hollow sphere around the scene
  const sr = 15 + Math.random() * 85;
  const sTheta = Math.random() * Math.PI * 2;
  const sPhi = Math.acos(2 * Math.random() - 1);

  starPositions[i * 3] = sr * Math.sin(sPhi) * Math.cos(sTheta);
  starPositions[i * 3 + 1] = sr * Math.sin(sPhi) * Math.sin(sTheta);
  starPositions[i * 3 + 2] = sr * Math.cos(sPhi);

  // Subtle color variation: warm whites, cool blues, faint purples
  const colorRand = Math.random();
  if (colorRand < 0.5) {
    // Warm golden white
    starColors[i * 3] = 0.95 + Math.random() * 0.05;
    starColors[i * 3 + 1] = 0.8 + Math.random() * 0.15;
    starColors[i * 3 + 2] = 0.6 + Math.random() * 0.2;
  } else if (colorRand < 0.8) {
    // Soft rose
    starColors[i * 3] = 0.9 + Math.random() * 0.1;
    starColors[i * 3 + 1] = 0.65 + Math.random() * 0.15;
    starColors[i * 3 + 2] = 0.7 + Math.random() * 0.2;
  } else {
    // Faint peach/coral accent
    starColors[i * 3] = 0.85 + Math.random() * 0.15;
    starColors[i * 3 + 1] = 0.5 + Math.random() * 0.2;
    starColors[i * 3 + 2] = 0.4 + Math.random() * 0.2;
  }

  starSizes[i] = Math.random() * 1.5 + 0.3;
}

starGeometry.setAttribute('position', new THREE.BufferAttribute(starPositions, 3));
starGeometry.setAttribute('color', new THREE.BufferAttribute(starColors, 3));
starGeometry.setAttribute('size', new THREE.BufferAttribute(starSizes, 1));

const starVertexShader = `
  attribute float size;
  varying vec3 vColor;
  varying float vBrightness;
  uniform float uTime;
  void main() {
    vColor = color;
    // Gentle twinkle based on position hash + time
    float twinkle = sin(uTime * 0.8 + position.x * 12.9898 + position.y * 78.233) * 0.5 + 0.5;
    twinkle = 0.5 + twinkle * 0.5;
    vBrightness = twinkle;
    vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
    gl_PointSize = size * twinkle * (60.0 / -mvPosition.z);
    gl_Position = projectionMatrix * mvPosition;
  }
`;

const starFragmentShader = `
  varying vec3 vColor;
  varying float vBrightness;
  void main() {
    float d = length(gl_PointCoord - vec2(0.5));
    if (d > 0.5) discard;
    float alpha = 1.0 - smoothstep(0.0, 0.5, d);
    alpha *= 0.6 * vBrightness;
    float glow = exp(-d * 8.0) * 0.2;
    vec3 col = vColor + glow;
    gl_FragColor = vec4(col, alpha);
  }
`;

const starMaterial = new THREE.ShaderMaterial({
  vertexShader: starVertexShader,
  fragmentShader: starFragmentShader,
  vertexColors: true,
  transparent: true,
  depthWrite: false,
  blending: THREE.AdditiveBlending,
  uniforms: {
    uTime: { value: 0.0 }
  }
});

const starField = new THREE.Points(starGeometry, starMaterial);
scene.add(starField);

// Soft atmospheric aura behind the sphere
const auraCanvas = document.createElement('canvas');
auraCanvas.width = 512;
auraCanvas.height = 512;
const auraCtx = auraCanvas.getContext('2d');
const auraGrad = auraCtx.createRadialGradient(256, 256, 0, 256, 256, 256);
auraGrad.addColorStop(0, 'rgba(255, 160, 80, 0.18)');
auraGrad.addColorStop(0.3, 'rgba(220, 80, 100, 0.08)');
auraGrad.addColorStop(0.6, 'rgba(140, 40, 60, 0.03)');
auraGrad.addColorStop(1, 'rgba(0, 0, 0, 0)');
auraCtx.fillStyle = auraGrad;
auraCtx.fillRect(0, 0, 512, 512);
const auraTexture = new THREE.CanvasTexture(auraCanvas);
const auraMaterial = new THREE.SpriteMaterial({
  map: auraTexture,
  transparent: true,
  blending: THREE.AdditiveBlending,
  depthWrite: false,
  opacity: 0.6,
});
const auraSprite = new THREE.Sprite(auraMaterial);
auraSprite.scale.set(6, 6, 1);
scene.add(auraSprite);

// Orbiting trail particles — small comets that spiral around the sphere during idle
const trailCount = 60;
const trailGeometry = new THREE.BufferGeometry();
const trailPositions = new Float32Array(trailCount * 3);
const trailColors = new Float32Array(trailCount * 3);
const trailSizes = new Float32Array(trailCount);
const trailData = []; // per-particle orbit parameters

for (let i = 0; i < trailCount; i++) {
  // Each trail particle has its own orbit
  const orbitRadius = 1.6 + Math.random() * 0.6;
  const orbitSpeed = 0.15 + Math.random() * 0.25;
  const orbitPhase = Math.random() * Math.PI * 2;
  const orbitTilt = (Math.random() - 0.5) * Math.PI * 0.6;
  const spiralDrift = (Math.random() - 0.5) * 0.08;
  const fadeIndex = i / trailCount; // 0..1, used for tail fade

  trailData.push({ orbitRadius, orbitSpeed, orbitPhase, orbitTilt, spiralDrift, fadeIndex });

  trailPositions[i * 3] = 0;
  trailPositions[i * 3 + 1] = 0;
  trailPositions[i * 3 + 2] = 0;

  // Amber fading to rose at tail
  const t = fadeIndex;
  trailColors[i * 3] = 0.95 - t * 0.15;
  trailColors[i * 3 + 1] = 0.55 - t * 0.2;
  trailColors[i * 3 + 2] = 0.25 + t * 0.4;

  trailSizes[i] = (1.0 - t * 0.6) * 2.0;
}

trailGeometry.setAttribute('position', new THREE.BufferAttribute(trailPositions, 3));
trailGeometry.setAttribute('color', new THREE.BufferAttribute(trailColors, 3));
trailGeometry.setAttribute('size', new THREE.BufferAttribute(trailSizes, 1));

const trailMaterial = new THREE.ShaderMaterial({
  vertexShader: `
    attribute float size;
    varying vec3 vColor;
    varying float vAlpha;
    uniform float uOpacity;
    void main() {
      vColor = color;
      vAlpha = uOpacity;
      vec4 mvPosition = modelViewMatrix * vec4(position, 1.0);
      gl_PointSize = size * (80.0 / -mvPosition.z);
      gl_Position = projectionMatrix * mvPosition;
    }
  `,
  fragmentShader: `
    varying vec3 vColor;
    varying float vAlpha;
    void main() {
      float d = length(gl_PointCoord - vec2(0.5));
      if (d > 0.5) discard;
      float alpha = 1.0 - smoothstep(0.05, 0.5, d);
      alpha *= 0.4 * vAlpha;
      float glow = exp(-d * 6.0) * 0.2;
      vec3 col = vColor + glow;
      gl_FragColor = vec4(col, alpha);
    }
  `,
  vertexColors: true,
  transparent: true,
  depthWrite: false,
  blending: THREE.AdditiveBlending,
  uniforms: {
    uOpacity: { value: 0.0 }
  }
});

const trailParticles = new THREE.Points(trailGeometry, trailMaterial);
scene.add(trailParticles);

// Faint nebula clouds using large transparent quads
const nebulaGroup = new THREE.Group();
scene.add(nebulaGroup);

function createNebulaCloud(color, pos, scale, opacity) {
  const canvas = document.createElement('canvas');
  canvas.width = 256;
  canvas.height = 256;
  const ctx = canvas.getContext('2d');
  const gradient = ctx.createRadialGradient(128, 128, 0, 128, 128, 128);
  gradient.addColorStop(0, `rgba(${color.r},${color.g},${color.b},${opacity})`);
  gradient.addColorStop(0.4, `rgba(${color.r},${color.g},${color.b},${opacity * 0.4})`);
  gradient.addColorStop(1, 'rgba(0,0,0,0)');
  ctx.fillStyle = gradient;
  ctx.fillRect(0, 0, 256, 256);

  const texture = new THREE.CanvasTexture(canvas);
  const nebMat = new THREE.SpriteMaterial({
    map: texture,
    transparent: true,
    blending: THREE.AdditiveBlending,
    depthWrite: false,
    opacity: 1.0
  });
  const sprite = new THREE.Sprite(nebMat);
  sprite.position.set(pos.x, pos.y, pos.z);
  sprite.scale.set(scale, scale, 1);
  return sprite;
}

// Scatter several faint nebula patches
const nebulaConfigs = [
  { color: { r: 120, g: 40, b: 20 }, pos: { x: -20, y: 10, z: -40 }, scale: 45, opacity: 0.08 },
  { color: { r: 100, g: 30, b: 50 }, pos: { x: 25, y: -8, z: -35 }, scale: 35, opacity: 0.06 },
  { color: { r: 80, g: 20, b: 40 }, pos: { x: -15, y: -15, z: -50 }, scale: 55, opacity: 0.05 },
  { color: { r: 90, g: 50, b: 15 }, pos: { x: 30, y: 20, z: -45 }, scale: 40, opacity: 0.07 },
  { color: { r: 110, g: 25, b: 45 }, pos: { x: -30, y: 5, z: -30 }, scale: 30, opacity: 0.06 },
  { color: { r: 70, g: 35, b: 25 }, pos: { x: 10, y: -20, z: -55 }, scale: 50, opacity: 0.04 },
];

nebulaConfigs.forEach(cfg => {
  nebulaGroup.add(createNebulaCloud(cfg.color, cfg.pos, cfg.scale, cfg.opacity));
});



// Audio setup for voice detection
let audioContext, analyser, dataArray, micActive = false;
let audioLevel = 0;
let smoothedLevel = 0;
let micStream = null;
let micActivationTime = -10;
let micDeactivationTime = -10;

async function setupAudio() {
  try {
    micStream = await navigator.mediaDevices.getUserMedia({ audio: true });
    audioContext = new AudioContext();
    const source = audioContext.createMediaStreamSource(micStream);
    analyser = audioContext.createAnalyser();
    analyser.fftSize = 256;
    analyser.smoothingTimeConstant = 0.8;
    source.connect(analyser);
    dataArray = new Uint8Array(analyser.frequencyBinCount);
    micActive = true;
    micActivationTime = clock.getElapsedTime();
    micButton.innerHTML = `${micIconOn}<span style="margin-left:8px">Listening</span>`;
    micButton.style.borderColor = 'rgba(255,160,80,0.3)';
    micButton.style.color = 'rgba(255,180,100,0.85)';
    statusDot.style.background = '#ffaa50';
  } catch (err) {
    console.warn('Mic access denied:', err);
    micButton.innerHTML = `${micIconMuted}<span style="margin-left:8px">Access denied</span>`;
    micButton.style.color = 'rgba(255,100,100,0.6)';
  }
}

function getAudioLevel() {
  if (!micActive || !analyser) return 0;
  analyser.getByteFrequencyData(dataArray);
  let sum = 0;
  for (let i = 0; i < dataArray.length; i++) {
    sum += dataArray[i];
  }
  return sum / (dataArray.length * 255);
}

// UI button
// Load Inter font
const fontLink = document.createElement('link');
fontLink.href = 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500&display=swap';
fontLink.rel = 'stylesheet';
document.head.appendChild(fontLink);

// Mic icon SVG (no emoji)
const micIconOff = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" y1="19" x2="12" y2="23"/><line x1="8" y1="23" x2="16" y2="23"/></svg>`;
const micIconOn = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 1a3 3 0 0 0-3 3v8a3 3 0 0 0 6 0V4a3 3 0 0 0-3-3z"/><path d="M19 10v2a7 7 0 0 1-14 0v-2"/><line x1="12" y1="19" x2="12" y2="23"/><line x1="8" y1="23" x2="16" y2="23"/></svg>`;
const micIconMuted = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><line x1="1" y1="1" x2="23" y2="23"/><path d="M9 9v3a3 3 0 0 0 5.12 2.12M15 9.34V4a3 3 0 0 0-5.94-.6"/><path d="M17 16.95A7 7 0 0 1 5 12v-2m14 0v2c0 .76-.13 1.49-.35 2.17"/><line x1="12" y1="19" x2="12" y2="23"/><line x1="8" y1="23" x2="16" y2="23"/></svg>`;

// Bottom control bar container
const controlBar = document.createElement('div');
controlBar.style.cssText = `
  position: fixed; bottom: 28px; left: 50%; transform: translateX(-50%);
  display: flex; align-items: center; gap: 12px;
  padding: 8px 12px; font-family: 'Inter', sans-serif;
  background: rgba(8, 8, 16, 0.75); border: 1px solid rgba(255,255,255,0.08);
  border-radius: 10px; z-index: 100; user-select: none;
  backdrop-filter: blur(12px); -webkit-backdrop-filter: blur(12px);
`;
document.body.appendChild(controlBar);

// Status dot
const statusDot = document.createElement('div');
statusDot.style.cssText = `
  width: 6px; height: 6px; border-radius: 50%;
  background: rgba(255,255,255,0.2); flex-shrink: 0;
  transition: background 0.4s;
`;
controlBar.appendChild(statusDot);

const micButton = document.createElement('div');
micButton.innerHTML = `${micIconOff}<span style="margin-left:8px">Enable microphone</span>`;
micButton.style.cssText = `
  display: flex; align-items: center; padding: 6px 14px;
  font-family: 'Inter', sans-serif; font-size: 12px; font-weight: 500;
  color: rgba(255,255,255,0.45); background: rgba(255,255,255,0.04);
  border: 1px solid rgba(255,255,255,0.08); border-radius: 6px;
  cursor: pointer; letter-spacing: 0.3px;
  transition: color 0.3s, border-color 0.3s, background 0.3s;
  line-height: 1;
`;
micButton.addEventListener('mouseenter', () => { micButton.style.background = 'rgba(255,255,255,0.07)'; micButton.style.borderColor = 'rgba(255,255,255,0.15)'; });
micButton.addEventListener('mouseleave', () => { micButton.style.background = 'rgba(255,255,255,0.04)'; micButton.style.borderColor = micActive ? 'rgba(255,160,80,0.3)' : 'rgba(255,255,255,0.08)'; });
controlBar.appendChild(micButton);
micButton.addEventListener('click', () => {
  if (!micActive) {
    setupAudio();
  } else {
    micActive = false;
    audioLevel = 0;
    smoothedLevel = 0;
    micDeactivationTime = clock.getElapsedTime();
    if (micStream) {
      micStream.getTracks().forEach(t => t.stop());
      micStream = null;
    }
    if (audioContext) {
      audioContext.close();
      audioContext = null;
      analyser = null;
      dataArray = null;
    }
    micButton.innerHTML = `${micIconMuted}<span style="margin-left:8px">Enable microphone</span>`;
    micButton.style.borderColor = 'rgba(255,255,255,0.08)';
    micButton.style.color = 'rgba(255,255,255,0.45)';
    statusDot.style.background = 'rgba(255,255,255,0.2)';
  }
});

// Level indicator inside control bar
const levelBar = document.createElement('div');
levelBar.style.cssText = `
  width: 80px; height: 3px; background: rgba(255,255,255,0.06);
  border-radius: 2px; overflow: hidden; flex-shrink: 0;
`;
const levelFill = document.createElement('div');
levelFill.style.cssText = `
  width: 0%; height: 100%; background: rgba(255, 160, 80, 0.55);
  border-radius: 2px; transition: width 0.05s;
`;
levelBar.appendChild(levelFill);
controlBar.appendChild(levelBar);



// Per-particle random offsets for organic scattering
const randomOffsets = new Float32Array(particleCount);
const randomPhases = new Float32Array(particleCount);
for (let i = 0; i < particleCount; i++) {
  randomOffsets[i] = Math.random() * 0.6 + 0.7;
  randomPhases[i] = Math.random() * Math.PI * 2;
}

// Clock for animation
const clock = new THREE.Clock();

// Animation loop
function animate() {
  requestAnimationFrame(animate);
  const elapsed = clock.getElapsedTime();

  // Get audio level
  audioLevel = getAudioLevel();
  smoothedLevel += (audioLevel - smoothedLevel) * 0.15;
  levelFill.style.width = `${Math.min(smoothedLevel * 300, 100)}%`;

  // Voice-reactive scatter strength (0 to ~2.5)
  const voiceScatter = smoothedLevel * 5.0;

  const posAttr = geometry.attributes.position;
  const posArray = posAttr.array;
  const sizeArray = geometry.attributes.size.array;
  const colorArray = geometry.attributes.color.array;
  const baseColors = window._baseColors;

  for (let i = 0; i < particleCount; i++) {
    const ox = originalPositions[i * 3];
    const oy = originalPositions[i * 3 + 1];
    const oz = originalPositions[i * 3 + 2];

    const r = Math.sqrt(ox * ox + oy * oy + oz * oz);
    const theta = Math.atan2(oz, ox);
    const phi = Math.acos(oy / r);

    // Base waves
    const wave1 = Math.sin(phi * 6 + elapsed * 1.2) * 0.15;
    const wave2 = Math.sin(theta * 4 + elapsed * 0.8) * 0.12;
    const wave3 = Math.sin(phi * 3 + theta * 5 + elapsed * 1.5) * 0.1;
    const wave4 = Math.sin(phi * 10 + theta * 8 + elapsed * 2.0) * 0.08;

    // Voice-driven scatter — each particle flies outward uniquely
    const voiceWave = Math.sin(phi * 7 + theta * 5 + elapsed * 3.0 + randomPhases[i]) * voiceScatter * randomOffsets[i];
    const voiceBurst = voiceScatter * randomOffsets[i] * 0.5;

    // Idle breathing — slow pulsing scale when mic is off
    const breathRate = 0.6;
    const breathDepth = micActive ? 0.01 : 0.04;
    const breathing = Math.sin(elapsed * breathRate) * breathDepth;

    // Idle particle drift — gentle individual wander when no voice input
    const idleAmount = micActive ? Math.max(0, 1.0 - voiceScatter * 2.0) : 1.0;
    const drift1 = Math.sin(elapsed * 0.4 + randomPhases[i] * 3.0) * 0.02 * idleAmount * randomOffsets[i];
    const drift2 = Math.cos(elapsed * 0.3 + randomPhases[i] * 2.0 + phi) * 0.02 * idleAmount * randomOffsets[i];
    const drift3 = Math.sin(elapsed * 0.5 + randomPhases[i] * 1.5 + theta) * 0.015 * idleAmount * randomOffsets[i];

    const displacement = 1 + wave1 + wave2 + wave3 + wave4 + voiceWave + voiceBurst + breathing;

    posArray[i * 3] = ox * displacement + drift1;
    posArray[i * 3 + 1] = oy * displacement + drift2;
    posArray[i * 3 + 2] = oz * displacement + drift3;

    // Particles grow slightly when voice is loud
    sizeArray[i] = (Math.random() * 0.3 + 0.5) + smoothedLevel * 4.0;
  }

  posAttr.needsUpdate = true;
  geometry.attributes.size.needsUpdate = true;

  // Mic activation ripple & color pulse
  const timeSinceActivation = elapsed - micActivationTime;

  // Reset colors to base before applying effects
  for (let i = 0; i < particleCount * 3; i++) {
    colorArray[i] = baseColors[i];
  }

  // Idle color shifting — slow hue cycle when mic is not active or voice is quiet
  const idleColorBlend = micActive ? Math.max(0, 1.0 - smoothedLevel * 8.0) * 0.5 : 1.0;
  if (idleColorBlend > 0.01) {
    const hueShift = elapsed * 0.08;
    for (let i = 0; i < particleCount; i++) {
      const oy = originalPositions[i * 3 + 1];
      const t = (oy / radius + 1) * 0.5;
      // Slowly rotate through amber → coral → rose → peach
      const shift = Math.sin(hueShift + t * 2.0) * 0.5 + 0.5;
      const shift2 = Math.sin(hueShift * 0.7 + t * 1.5 + 1.0) * 0.5 + 0.5;
      colorArray[i * 3]     = baseColors[i * 3] + shift * 0.08 * idleColorBlend;
      colorArray[i * 3 + 1] = baseColors[i * 3 + 1] * (1.0 - idleColorBlend * 0.15) + shift2 * 0.06 * idleColorBlend;
      colorArray[i * 3 + 2] = baseColors[i * 3 + 2] * (1.0 - idleColorBlend * 0.2) + shift * 0.1 * idleColorBlend;
    }
    geometry.attributes.color.needsUpdate = true;
  }

  // Mouse repulsion — project mouse into 3D space and repel nearby particles
  raycaster.setFromCamera(mouseNDC, camera);
  // Place the mouse target on a plane at the sphere's center distance
  const planeNormal = new THREE.Vector3(0, 0, 1).applyQuaternion(camera.quaternion);
  const plane = new THREE.Plane(planeNormal, 0);
  const mouseRay = raycaster.ray;
  const intersectPoint = new THREE.Vector3();
  mouseRay.intersectPlane(plane, intersectPoint);

  if (intersectPoint && mouseNDC.x !== 9999) {
    mouseWorld.copy(intersectPoint);

    // Transform mouseWorld into the particles' local space (accounting for rotation)
    const invMatrix = new THREE.Matrix4().copy(particles.matrixWorld).invert();
    const localMouse = mouseWorld.clone().applyMatrix4(invMatrix);

    for (let i = 0; i < particleCount; i++) {
      const px = posArray[i * 3];
      const py = posArray[i * 3 + 1];
      const pz = posArray[i * 3 + 2];

      const dx = px - localMouse.x;
      const dy = py - localMouse.y;
      const dz = pz - localMouse.z;
      const dist = Math.sqrt(dx * dx + dy * dy + dz * dz);

      if (dist < mouseInfluenceRadius && dist > 0.001) {
        const falloff = 1.0 - (dist / mouseInfluenceRadius);
        const strength = falloff * falloff * mouseRepelStrength;
        const nx = dx / dist;
        const ny = dy / dist;
        const nz = dz / dist;
        posArray[i * 3] += nx * strength;
        posArray[i * 3 + 1] += ny * strength;
        posArray[i * 3 + 2] += nz * strength;

        // Brighten particles near mouse with warm glow
        const glow = falloff * 0.6;
        colorArray[i * 3] = Math.min(1.0, colorArray[i * 3] + glow * 0.8);
        colorArray[i * 3 + 1] = Math.min(1.0, colorArray[i * 3 + 1] + glow * 0.5);
        colorArray[i * 3 + 2] = Math.min(1.0, colorArray[i * 3 + 2] + glow * 0.2);
      }
    }
    geometry.attributes.color.needsUpdate = true;
  }

  if (timeSinceActivation < 2.5) {
    const rippleSpeed = 3.0;
    const rippleFront = timeSinceActivation * rippleSpeed;
    const rippleWidth = 0.8;

    for (let i = 0; i < particleCount; i++) {
      const ox = originalPositions[i * 3];
      const oy = originalPositions[i * 3 + 1];
      const oz = originalPositions[i * 3 + 2];
      const dist = Math.sqrt(ox * ox + oy * oy + oz * oz);

      // Ripple intensity based on distance from expanding wavefront
      const rippleDist = Math.abs(dist - rippleFront);
      const ripple = Math.max(0, 1.0 - rippleDist / rippleWidth);
      const pulse = ripple * Math.max(0, 1.0 - timeSinceActivation / 2.5);

      // Flash bright warm white then fade back to base color
      colorArray[i * 3] = baseColors[i * 3] + (1.0 - baseColors[i * 3]) * pulse;
      colorArray[i * 3 + 1] = baseColors[i * 3 + 1] + (0.7 - baseColors[i * 3 + 1]) * pulse;
      colorArray[i * 3 + 2] = baseColors[i * 3 + 2] + (0.2) * pulse;

      // Ripple also nudges particles outward briefly
      const ripplePush = pulse * 0.3;
      posArray[i * 3] += (ox / dist) * ripplePush;
      posArray[i * 3 + 1] += (oy / dist) * ripplePush;
      posArray[i * 3 + 2] += (oz / dist) * ripplePush;
    }
    geometry.attributes.color.needsUpdate = true;
    posAttr.needsUpdate = true;
  } else if (timeSinceActivation < 3.0) {
    // Ensure colors return to base after ripple ends
    for (let i = 0; i < particleCount * 3; i++) {
      colorArray[i] = baseColors[i];
    }
    geometry.attributes.color.needsUpdate = true;
  }

  // Mic deactivation contracting ripple — inward with dimming color
  const timeSinceDeactivation = elapsed - micDeactivationTime;
  if (timeSinceDeactivation < 2.5) {
    const maxRadius = 5.0;
    const rippleSpeed = 2.5;
    const rippleFront = maxRadius - timeSinceDeactivation * rippleSpeed;
    const rippleWidth = 0.9;

    for (let i = 0; i < particleCount; i++) {
      const ox = originalPositions[i * 3];
      const oy = originalPositions[i * 3 + 1];
      const oz = originalPositions[i * 3 + 2];
      const dist = Math.sqrt(ox * ox + oy * oy + oz * oz);

      // Contracting ripple intensity
      const rippleDist = Math.abs(dist - Math.max(0, rippleFront));
      const ripple = Math.max(0, 1.0 - rippleDist / rippleWidth);
      const fade = Math.max(0, 1.0 - timeSinceDeactivation / 2.5);
      const pulse = ripple * fade;

      // Dim toward dark blue/purple
      colorArray[i * 3] = baseColors[i * 3] * (1.0 - pulse * 0.7);
      colorArray[i * 3 + 1] = baseColors[i * 3 + 1] * (1.0 - pulse * 0.8);
      colorArray[i * 3 + 2] = baseColors[i * 3 + 2] * (1.0 - pulse * 0.3) + pulse * 0.15;

      // Pull particles inward slightly as wave passes
      const ripplePull = pulse * 0.2;
      posArray[i * 3] -= (ox / dist) * ripplePull;
      posArray[i * 3 + 1] -= (oy / dist) * ripplePull;
      posArray[i * 3 + 2] -= (oz / dist) * ripplePull;
    }
    geometry.attributes.color.needsUpdate = true;
    posAttr.needsUpdate = true;
  } else if (timeSinceDeactivation >= 2.5 && timeSinceDeactivation < 3.0) {
    // Restore base colors after deactivation ripple
    for (let i = 0; i < particleCount * 3; i++) {
      colorArray[i] = baseColors[i];
    }
    geometry.attributes.color.needsUpdate = true;
  }



  // Smooth mouse-driven orbit tilt
  mouseOrbitCurrent.x += (mouseOrbitTarget.x - mouseOrbitCurrent.x) * mouseOrbitSmoothing;
  mouseOrbitCurrent.y += (mouseOrbitTarget.y - mouseOrbitCurrent.y) * mouseOrbitSmoothing;
  particles.rotation.x = mouseOrbitCurrent.x;

  // Slow rotation + mouse horizontal influence
  particles.rotation.y += 0.001;
  particles.rotation.y += (mouseOrbitCurrent.y - particles.rotation.y) * 0.01;

  // Mouse press scale — smooth spring toward target
  const sphereScaleTarget = mousePressed ? sphereScalePressed : 1.0;
  sphereScaleCurrent += (sphereScaleTarget - sphereScaleCurrent) * sphereScaleSmoothing;
  particles.scale.set(sphereScaleCurrent, sphereScaleCurrent, sphereScaleCurrent);

  // Update star twinkle
  starMaterial.uniforms.uTime.value = elapsed;

  // Update trail particles — orbit/spiral during idle, fade out when mic active
  const trailTargetOpacity = micActive ? Math.max(0, 0.5 - smoothedLevel * 4.0) : 1.0;
  const currentTrailOpacity = trailMaterial.uniforms.uOpacity.value;
  trailMaterial.uniforms.uOpacity.value += (trailTargetOpacity - currentTrailOpacity) * 0.03;

  const trailPosArray = trailGeometry.attributes.position.array;
  const trailSizeArray = trailGeometry.attributes.size.array;

  // Group trail particles into streams (5 streams of 12 particles each)
  const streamsCount = 5;
  const perStream = trailCount / streamsCount;

  for (let s = 0; s < streamsCount; s++) {
    for (let j = 0; j < perStream; j++) {
      const i = s * perStream + j;
      const d = trailData[i];

      // Each particle in the stream is offset in time to create a trailing tail
      const tailDelay = j * 0.12;
      const t = elapsed * d.orbitSpeed - tailDelay;

      // Spiral radius slowly drifts outward and wraps
      const spiralR = d.orbitRadius + Math.sin(elapsed * 0.1 + d.orbitPhase) * 0.15 + d.spiralDrift * Math.sin(t);

      // 3D orbit with tilt
      const cosT = Math.cos(d.orbitTilt);
      const sinT = Math.sin(d.orbitTilt);
      const angle = t + d.orbitPhase;

      const lx = spiralR * Math.cos(angle);
      const ly = spiralR * Math.sin(angle) * cosT;
      const lz = spiralR * Math.sin(angle) * sinT;

      trailPosArray[i * 3] = lx;
      trailPosArray[i * 3 + 1] = ly;
      trailPosArray[i * 3 + 2] = lz;

      // Tail particles get smaller
      const tailFade = 1.0 - (j / perStream);
      trailSizeArray[i] = tailFade * tailFade * 2.2;
    }
  }

  trailGeometry.attributes.position.needsUpdate = true;
  trailGeometry.attributes.size.needsUpdate = true;

  // Aura breathing — pulsing glow behind the sphere
  const auraPulse = micActive
    ? 0.6 + smoothedLevel * 2.0
    : 0.4 + Math.sin(elapsed * 0.6) * 0.15;
  auraMaterial.opacity = Math.min(1.0, auraPulse);
  const auraScale = micActive
    ? 6 + smoothedLevel * 4.0
    : 5.5 + Math.sin(elapsed * 0.6) * 0.5;
  auraSprite.scale.set(auraScale, auraScale, 1);

  // Very slow drift on star field and nebula
  starField.rotation.y += 0.00008;
  starField.rotation.x += 0.00003;
  nebulaGroup.rotation.y += 0.00005;



  controls.update();
  renderer.render(scene, camera);
}
animate();

// Handle resize
window.addEventListener('resize', () => {
  camera.aspect = window.innerWidth / window.innerHeight;
  camera.updateProjectionMatrix();
  renderer.setSize(window.innerWidth, window.innerHeight);
});