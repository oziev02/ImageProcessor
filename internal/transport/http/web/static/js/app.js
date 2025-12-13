const API_BASE = window.location.origin;
let refreshInterval;

// File input handler
document.getElementById('fileInput').addEventListener('change', function(e) {
    const fileName = e.target.files[0]?.name || '';
    document.getElementById('fileName').textContent = fileName;
});

// Upload form handler
document.getElementById('uploadForm').addEventListener('submit', async function(e) {
    e.preventDefault();
    
    const fileInput = document.getElementById('fileInput');
    const file = fileInput.files[0];
    
    if (!file) {
        showMessage('Пожалуйста, выберите файл', 'error');
        return;
    }

    const formData = new FormData();
    formData.append('image', file);

    const uploadBtn = document.getElementById('uploadBtn');
    uploadBtn.disabled = true;
    uploadBtn.textContent = 'Загрузка...';

    try {
        const response = await fetch(API_BASE + '/upload', {
            method: 'POST',
            body: formData
        });

        if (!response.ok) {
            throw new Error('Ошибка загрузки');
        }

        const image = await response.json();
        showMessage('Изображение успешно загружено! Обработка началась...', 'success');
        fileInput.value = '';
        document.getElementById('fileName').textContent = '';
        
        // Refresh images list
        loadImages();
    } catch (error) {
        showMessage('Ошибка при загрузке: ' + error.message, 'error');
    } finally {
        uploadBtn.disabled = false;
        uploadBtn.textContent = 'Загрузить и обработать';
    }
});

// Load images
async function loadImages() {
    try {
        const response = await fetch(API_BASE + '/api/images?limit=50');
        if (!response.ok) throw new Error('Ошибка загрузки');
        
        const images = await response.json();
        renderImages(images);
    } catch (error) {
        document.getElementById('imagesContainer').innerHTML = 
            '<div class="error">Ошибка загрузки изображений: ' + error.message + '</div>';
    }
}

// Render images
function renderImages(images) {
    const container = document.getElementById('imagesContainer');
    
    if (images.length === 0) {
        container.innerHTML = '<div class="loading">Нет загруженных изображений</div>';
        return;
    }

    const hasProcessing = images.some(img => img.status === 'pending' || img.status === 'processing');
    
    container.innerHTML = '<div class="images-grid">' + 
        images.map(img => 
            '<div class="image-card">' +
                '<div class="image-preview">' +
                    getImagePreview(img) +
                '</div>' +
                '<div class="image-info">' +
                    '<div class="image-status status-' + img.status + '">' + getStatusText(img.status) + '</div>' +
                    '<div class="image-id">ID: ' + img.id + '</div>' +
                    '<div class="image-dimensions">' +
                        img.original_width + ' × ' + img.original_height + 'px' +
                        (img.processed_width ? ' → ' + img.processed_width + ' × ' + img.processed_height + 'px' : '') +
                    '</div>' +
                    '<button class="delete-btn" onclick="deleteImage(\'' + img.id + '\')">Удалить</button>' +
                '</div>' +
            '</div>'
        ).join('') + '</div>';

    // Auto-refresh if there are processing images
    if (hasProcessing) {
        if (!refreshInterval) {
            refreshInterval = setInterval(loadImages, 2000);
        }
    } else {
        if (refreshInterval) {
            clearInterval(refreshInterval);
            refreshInterval = null;
        }
    }
}

// Get image preview
function getImagePreview(img) {
    if (img.status === 'completed' && img.processed_path) {
        return '<img src="' + API_BASE + '/image/' + img.id + '" alt="Processed" style="width: 100%; height: 100%; object-fit: cover;">';
    } else if (img.status === 'processing' || img.status === 'pending') {
        return '<div style="text-align: center; color: #999;">⏳ Обработка...</div>';
    } else if (img.status === 'failed') {
        return '<div style="text-align: center; color: #dc3545;">❌ Ошибка обработки</div>';
    } else {
        return '<div style="text-align: center; color: #999;">📷 Изображение</div>';
    }
}

// Get status text
function getStatusText(status) {
    const statusMap = {
        'pending': 'Ожидание',
        'processing': 'Обработка',
        'completed': 'Готово',
        'failed': 'Ошибка'
    };
    return statusMap[status] || status;
}

// Delete image
async function deleteImage(id) {
    if (!confirm('Удалить это изображение?')) return;

    try {
        const response = await fetch(API_BASE + '/image/' + id, {
            method: 'DELETE'
        });

        if (!response.ok) throw new Error('Ошибка удаления');

        showMessage('Изображение удалено', 'success');
        loadImages();
    } catch (error) {
        showMessage('Ошибка при удалении: ' + error.message, 'error');
    }
}

// Show message
function showMessage(text, type) {
    const messageDiv = document.getElementById('message');
    messageDiv.className = type;
    messageDiv.textContent = text;
    messageDiv.style.display = 'block';
    
    setTimeout(() => {
        messageDiv.style.display = 'none';
    }, 5000);
}

// Initial load
loadImages();

