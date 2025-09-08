#!/bin/bash

# Script para automatizar carga y descarga de módulos de kernel
# Autor: Sistema de gestión de módulos
# Uso: ./script.sh [load|unload|reload|status]

# Configuración de rutas de módulos
SYSINFO_MODULE="/home/abdielo/dev/sopes/proyecto1/proyecto-2/modulo-kernel/sysinfo/sysinfo_so1.ko"
CONTINFO_MODULE="/home/abdielo/dev/sopes/proyecto1/proyecto-2/modulo-kernel/continfo/continfo_so1.ko"

# Nombres de los módulos (sin extensión .ko)
SYSINFO_NAME="sysinfo_so1"
CONTINFO_NAME="continfo_so1"

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Función para logging
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Función para verificar si un módulo está cargado
is_module_loaded() {
    lsmod | grep -q "^$1 "
    return $?
}

# Función para verificar si el archivo del módulo existe
check_module_file() {
    if [ ! -f "$1" ]; then
        log_error "Archivo de módulo no encontrado: $1"
        return 1
    fi
    return 0
}

# Función para cargar un módulo
load_module() {
    local module_path=$1
    local module_name=$2
    
    if ! check_module_file "$module_path"; then
        return 1
    fi
    
    if is_module_loaded "$module_name"; then
        log_warn "Módulo $module_name ya está cargado"
        return 0
    fi
    
    log_info "Cargando módulo: $module_name"
    log_info "Ruta del módulo: $module_path"
    
    # Verificar permisos del archivo
    if [ ! -r "$module_path" ]; then
        log_error "No se puede leer el archivo del módulo: $module_path"
        return 1
    fi
    
    # Intentar cargar con información detallada de errores
    local insmod_output
    if insmod_output=$(sudo insmod "$module_path" 2>&1); then
        log_info "Módulo $module_name cargado exitosamente"
        return 0
    else
        log_error "Error al cargar módulo $module_name"
        log_error "Salida de insmod: $insmod_output"
        
        # Mostrar información adicional
        log_error "Información del archivo:"
        ls -la "$module_path" 2>/dev/null || log_error "No se puede acceder al archivo"
        
        # Verificar información del módulo
        log_error "Información del módulo:"
        modinfo "$module_path" 2>/dev/null | head -10 || log_error "No se puede obtener modinfo"
        
        # Verificar compatibilidad de kernel
        local kernel_version=$(uname -r)
        local module_kernel=$(modinfo "$module_path" 2>/dev/null | grep vermagic | cut -d: -f2 | xargs)
        log_error "Kernel actual: $kernel_version"
        log_error "Kernel del módulo: $module_kernel"
        
        # Verificar logs del kernel
        log_error "Últimas líneas de dmesg:"
        sudo dmesg | tail -5 2>/dev/null || log_error "No se puede acceder a dmesg"
        
        return 1
    fi
}

# Función para descargar un módulo
unload_module() {
    local module_name=$1
    
    if ! is_module_loaded "$module_name"; then
        log_warn "Módulo $module_name no está cargado"
        return 0
    fi
    
    log_info "Descargando módulo: $module_name"
    if sudo rmmod "$module_name"; then
        log_info "Módulo $module_name descargado exitosamente"
        return 0
    else
        log_error "Error al descargar módulo $module_name"
        return 1
    fi
}

# Función para cargar todos los módulos
load_all() {
    log_info "Cargando todos los módulos..."
    local success=0
    
    load_module "$SYSINFO_MODULE" "$SYSINFO_NAME" || success=1
    load_module "$CONTINFO_MODULE" "$CONTINFO_NAME" || success=1
    
    if [ $success -eq 0 ]; then
        log_info "Todos los módulos cargados exitosamente"
    else
        log_error "Algunos módulos no pudieron cargarse"
    fi
    
    return $success
}

# Función para descargar todos los módulos
unload_all() {
    log_info "Descargando todos los módulos..."
    local success=0
    
    # Descarga en orden inverso por si hay dependencias
    unload_module "$CONTINFO_NAME" || success=1
    unload_module "$SYSINFO_NAME" || success=1
    
    if [ $success -eq 0 ]; then
        log_info "Todos los módulos descargados exitosamente"
    else
        log_error "Algunos módulos no pudieron descargarse"
    fi
    
    return $success
}

# Función para recargar todos los módulos
reload_all() {
    log_info "Recargando todos los módulos..."
    unload_all
    sleep 1  # Pequeña pausa para asegurar la descarga completa
    load_all
}

# Función para mostrar el estado de los módulos
show_status() {
    echo "==============================================="
    echo "          Estado de Módulos de Kernel"
    echo "==============================================="
    
    printf "%-20s %-10s %-30s\n" "MÓDULO" "ESTADO" "ARCHIVO"
    echo "-----------------------------------------------"
    
    # Verificar sysinfo
    if is_module_loaded "$SYSINFO_NAME"; then
        status="CARGADO"
        color=$GREEN
    else
        status="NO CARGADO"
        color=$RED
    fi
    printf "%-20s ${color}%-10s${NC} %-30s\n" "$SYSINFO_NAME" "$status" "$(basename $SYSINFO_MODULE)"
    
    # Verificar continfo
    if is_module_loaded "$CONTINFO_NAME"; then
        status="CARGADO"
        color=$GREEN
    else
        status="NO CARGADO"
        color=$RED
    fi
    printf "%-20s ${color}%-10s${NC} %-30s\n" "$CONTINFO_NAME" "$status" "$(basename $CONTINFO_MODULE)"
    
    echo "==============================================="
}

# Función para mostrar ayuda
show_help() {
    echo "Uso: $0 [comando]"
    echo ""
    echo "Comandos disponibles:"
    echo "  load     - Cargar todos los módulos de kernel"
    echo "  unload   - Descargar todos los módulos de kernel"
    echo "  reload   - Recargar todos los módulos de kernel"
    echo "  status   - Mostrar estado actual de los módulos"
    echo "  help     - Mostrar esta ayuda"
    echo ""
    echo "Si no se especifica comando, se ejecutará 'load' por defecto"
    echo ""
    echo "Módulos gestionados:"
    echo "  - $SYSINFO_NAME"
    echo "  - $CONTINFO_NAME"
}

# Función principal
main() {
    local command=${1:-load}  # Por defecto cargar módulos
    
    case "$command" in
        "load")
            load_all
            ;;
        "unload")
            unload_all
            ;;
        "reload")
            reload_all
            ;;
        "status")
            show_status
            ;;
        "help"|"-h"|"--help")
            show_help
            ;;
        *)
            log_error "Comando desconocido: $command"
            show_help
            exit 1
            ;;
    esac
}

# Verificar que el script se ejecute con los permisos adecuados
if [ "$EUID" -eq 0 ]; then
    log_warn "Ejecutándose como root. Se recomienda usar sudo configurado correctamente."
fi

# Ejecutar función principal
main "$@"